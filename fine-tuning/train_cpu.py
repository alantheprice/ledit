#!/usr/bin/env python3
"""
Fine-tune Phi-3 model for security validation on CPU.

WARNING: CPU training is VERY SLOW (hours to days).
Use train_4090.py for GPU training (5-10 minutes).

This script is provided as a fallback for systems without GPU access.
It uses CPU-optimized settings but will still be slow.

Requirements:
    pip install torch transformers datasets accelerate

Usage:
    python3 train_cpu.py

Expected Results:
    - Training time: Several hours to days (depends on CPU)
    - CPU utilization: ~100% across cores
    - Accuracy: >90%

Note: For production use, train on RTX 4090 using train_4090.py instead.
"""

import os
import json
import torch
from transformers import (
    AutoTokenizer,
    AutoModelForCausalLM,
    TrainingArguments,
    Trainer,
    DataCollatorForLanguageModeling
)
from datasets import Dataset

# Configuration - using a freely available model for CPU
MODEL_NAME = "microsoft/Phi-3-mini-4k-instruct"  # 3.8B model, open access
OUTPUT_DIR = "./checkpoints_cpu"
MAX_LENGTH = 256  # Shorter sequences for CPU
BATCH_SIZE = 2  # Very small batch for CPU
NUM_EPOCHS = 2  # Fewer epochs for faster training
LEARNING_RATE = 1e-4
GRADIENT_ACCUMULATION_STEPS = 8  # Effective batch = 2 * 8 = 16

def load_jsonl(file_path):
    with open(file_path, 'r') as f:
        return [json.loads(line) for line in f]

def format_prompt(example):
    prompt = example['prompt']
    completion = example['completion']
    return f"{prompt}\n\n{completion}"

def prepare_dataset(examples, tokenizer):
    texts = [format_prompt(ex) for ex in examples]

    encodings = tokenizer(
        texts,
        truncation=True,
        max_length=MAX_LENGTH,
        padding="max_length",
        return_tensors="pt"
    )

    dataset = Dataset.from_dict({
        'input_ids': encodings['input_ids'],
        'attention_mask': encodings['attention_mask'],
        'labels': encodings['input_ids'].clone()
    })

    return dataset

def main():
    print("=" * 60)
    print("CPU Fine-tuning for Security Validation")
    print("=" * 60)
    print(f"📁 Model: {MODEL_NAME}")
    print(f"💻 Device: CPU")
    print(f"⚡  Using smaller dataset for faster training (200 examples)")
    print()

    # Load model
    print("📦 Loading model (this may take a few minutes)...")
    tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME, trust_remote_code=True)
    model = AutoModelForCausalLM.from_pretrained(
        MODEL_NAME,
        torch_dtype=torch.float32,
        trust_remote_code=True,
        low_cpu_mem_usage=True,
    )

    if tokenizer.pad_token is None:
        tokenizer.pad_token = tokenizer.eos_token
        model.config.pad_token_id = tokenizer.eos_token_id

    # Load training data
    print("📊 Loading training data...")
    train_examples = load_jsonl('output/train_small.jsonl')
    print(f"  Training examples: {len(train_examples)}")
    print()

    # Prepare dataset
    print("🔧 Preparing dataset...")
    train_dataset = prepare_dataset(train_examples, tokenizer)

    # Training arguments optimized for CPU
    print("⚙️  Configuring training...")
    training_args = TrainingArguments(
        output_dir=OUTPUT_DIR,
        overwrite_output_dir=True,
        num_train_epochs=NUM_EPOCHS,
        per_device_train_batch_size=BATCH_SIZE,
        gradient_accumulation_steps=GRADIENT_ACCUMULATION_STEPS,
        learning_rate=LEARNING_RATE,
        warmup_steps=20,
        logging_steps=5,
        save_steps=100,
        save_total_limit=2,
        prediction_loss_only=True,
        remove_unused_columns=False,
        fp16=False,
        bf16=False,
        save_safetensors=False,
        report_to="none",
        skip_memory_metrics=True,
    )

    data_collator = DataCollatorForLanguageModeling(
        tokenizer=tokenizer,
        mlm=False,
    )

    trainer = Trainer(
        model=model,
        args=training_args,
        train_dataset=train_dataset,
        data_collator=data_collator,
    )

    # Train
    print()
    print("🎯 Starting training...")
    print(f"  Epochs: {NUM_EPOCHS}")
    print(f"  Batch size: {BATCH_SIZE}")
    print(f"  Gradient accumulation: {GRADIENT_ACCUMULATION_STEPS}")
    print(f"  Effective batch size: {BATCH_SIZE * GRADIENT_ACCUMULATION_STEPS}")
    print()
    print("⏳ This will take approximately 30-60 minutes on CPU...")
    print()

    train_result = trainer.train()

    # Save model
    print()
    print("💾 Saving fine-tuned model...")
    trainer.save_model("./ledit-security-cpu")
    tokenizer.save_pretrained("./ledit-security-cpu")

    # Print metrics
    print()
    print("✅ Training complete!")
    print()
    print("📊 Final metrics:")
    for key, value in train_result.metrics.items():
        if isinstance(value, (int, float)):
            print(f"  {key}: {value:.4f}")

    print()
    print(f"📁 Model saved to: ./ledit-security-cpu")
    print()
    print("To test the model:")
    print("  python3 -m transformers.pipeline text-generation ./ledit-security-cpu")

if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""
Fine-tune Phi-3 model for security validation on Apple M3 Max (MPS).

WARNING: This script has known compatibility issues with Phi-3 on MPS.
Use train_4090.py for GPU training (5-10 minutes) instead.

This script uses Apple's Metal Performance Shaders (MPS) for GPU
acceleration on Apple Silicon Macs.

Known Issues:
    - Phi-3 model has compatibility issues with MPS backend
    - DynamicCache errors may occur
    - Consider using CPU training (train_cpu.py) as fallback

Requirements:
    pip install torch transformers datasets accelerate

Usage:
    python3 train_mps.py

Expected Results:
    - Training time: 10-20 minutes on M3 Max
    - GPU utilization: ~60-80%
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

# Configuration - optimized for M3 Max
MODEL_NAME = "microsoft/Phi-3-mini-4k-instruct"  # 3.8B model
OUTPUT_DIR = "./checkpoints_mps"
MAX_LENGTH = 512
BATCH_SIZE = 8  # Larger batch for GPU
NUM_EPOCHS = 3
LEARNING_RATE = 5e-5
GRADIENT_ACCUMULATION_STEPS = 1  # No accumulation needed on GPU

def load_jsonl(file_path):
    with open(file_path, 'r') as f:
        return [json.loads(line) for line in f]

def format_prompt(example):
    prompt = example['prompt']
    completion = example['completion']
    return f"{prompt}\n\n{completion}"

def prepare_dataset(examples, tokenizer):
    print(f"Tokenizing {len(examples)} examples...")
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
    print("MPS Fine-tuning for Security Validation")
    print("=" * 60)
    print(f"📁 Model: {MODEL_NAME}")
    print(f"💻 Device: MPS (Apple M3 Max GPU)")
    print(f"⚡  GPU Acceleration: Enabled")
    print()

    # Check MPS availability
    if not torch.backends.mps.is_available():
        print("❌ MPS is not available!")
        return

    device = torch.device("mps")
    print(f"✅ Using device: {device}")
    print()

    # Load model with MPS optimization
    print("📦 Loading model...")
    tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME, trust_remote_code=True)
    model = AutoModelForCausalLM.from_pretrained(
        MODEL_NAME,
        torch_dtype=torch.float32,  # MPS works best with float32
        trust_remote_code=True,
    ).to(device)  # Move to MPS device

    if tokenizer.pad_token is None:
        tokenizer.pad_token = tokenizer.eos_token
        model.config.pad_token_id = tokenizer.eos_token_id

    print(f"✅ Model loaded on {device}")
    print()

    # Load training data
    print("📊 Loading training data...")
    train_examples = load_jsonl('output/train.jsonl')
    print(f"  Training examples: {len(train_examples)}")
    print()

    # Prepare dataset
    print("🔧 Preparing dataset...")
    train_dataset = prepare_dataset(train_examples, tokenizer)
    print(f"✅ Dataset prepared")
    print()

    # Training arguments optimized for MPS
    print("⚙️  Configuring training for MPS...")
    training_args = TrainingArguments(
        output_dir=OUTPUT_DIR,
        overwrite_output_dir=True,
        num_train_epochs=NUM_EPOCHS,
        per_device_train_batch_size=BATCH_SIZE,
        gradient_accumulation_steps=GRADIENT_ACCUMULATION_STEPS,
        learning_rate=LEARNING_RATE,
        warmup_steps=50,
        logging_steps=10,
        save_steps=100,
        eval_steps=100,
        save_total_limit=2,
        prediction_loss_only=True,
        remove_unused_columns=False,
        fp16=False,  # MPS doesn't support FP16
        bf16=False,
        save_safetensors=False,
        report_to="none",
        # Let transformers auto-detect MPS
        ddp_find_unused_parameters=False,
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
    print("🎯 Starting GPU-accelerated training...")
    print(f"  Epochs: {NUM_EPOCHS}")
    print(f"  Batch size: {BATCH_SIZE}")
    print(f"  Device: MPS (M3 Max GPU)")
    print(f"  Learning rate: {LEARNING_RATE}")
    print()
    print("⏱️  Estimated time: 10-20 minutes on M3 Max")
    print()

    import time
    start_time = time.time()

    train_result = trainer.train()

    elapsed = time.time() - start_time
    print()
    print(f"⏱️  Training completed in {elapsed/60:.1f} minutes")

    # Save model
    print()
    print("💾 Saving fine-tuned model...")
    model_save_path = "./ledit-security-mps"
    trainer.save_model(model_save_path)
    tokenizer.save_pretrained(model_save_path)

    print(f"✅ Model saved to: {model_save_path}")

    # Print metrics
    print()
    print("📊 Final metrics:")
    for key, value in train_result.metrics.items():
        if isinstance(value, (int, float)):
            print(f"  {key}: {value:.4f}")

    print()
    print("=" * 60)
    print("✅ Fine-tuning complete!")
    print("=" * 60)
    print()
    print("Next steps to use the model:")
    print("  1. Convert to GGUF for Ollama")
    print("  2. Or test directly: python3 test_mps_model.py")
    print()

if __name__ == "__main__":
    main()

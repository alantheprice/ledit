#!/usr/bin/env python3
"""
Fine-tune Phi-3 model for security validation on NVIDIA RTX 4090.

This script fine-tunes microsoft/Phi-3-mini-4k-instruct (3.8B parameters)
on a custom security validation dataset to classify shell commands and file
operations as SAFE (0), CAUTION (1), or DANGEROUS (2).

Requirements:
    pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu118
    pip install transformers datasets accelerate

Usage:
    python3 train_4090.py

Expected Results:
    - Training time: 5-10 minutes on RTX 4090
    - GPU utilization: ~70-90%
    - VRAM usage: ~12-16 GB / 24 GB
    - Accuracy: >90% (up from 67% with off-the-shelf models)

Output:
    - ./ledit-security-4090/ - Fine-tuned model in HuggingFace format
    - ./checkpoints_4090/ - Training checkpoints

Next Steps (see README_4090.md for details):
    1. Test the model with security validation test suite
    2. Convert to GGUF format for Ollama deployment
    3. Upload to Hugging Face for distribution
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

# Configuration - optimized for RTX 4090
MODEL_NAME = "microsoft/Phi-3-mini-4k-instruct"  # 3.8B model
OUTPUT_DIR = "./checkpoints_4090"
MAX_LENGTH = 512
BATCH_SIZE = 16  # Large batch for 4090
NUM_EPOCHS = 3
LEARNING_RATE = 5e-5
GRADIENT_ACCUMULATION_STEPS = 1

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
    print("CUDA Fine-tuning for Security Validation")
    print("=" * 60)
    print(f"📁 Model: {MODEL_NAME}")
    print(f"💻 Device: CUDA (NVIDIA RTX 4090)")
    print(f"⚡  GPU: {torch.cuda.get_device_name(0) if torch.cuda.is_available() else 'N/A'}")
    print(f"📊 VRAM: {torch.cuda.get_device_properties(0).total_memory / 1024**3:.1f} GB")
    print()

    # Check CUDA availability
    if not torch.cuda.is_available():
        print("❌ CUDA is not available!")
        print("Please ensure you're running this on the machine with the 4090")
        return

    device = torch.device("cuda")
    print(f"✅ Using device: {device}")
    print()

    # Load model with optimizations
    print("📦 Loading model...")
    tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME, trust_remote_code=True)

    # Load model with bfloat16 for speed and memory efficiency
    model = AutoModelForCausalLM.from_pretrained(
        MODEL_NAME,
        torch_dtype=torch.bfloat16,  # 4090 supports BF16
        trust_remote_code=True,
        device_map="auto",  # Automatically map to GPU
    )

    if tokenizer.pad_token is None:
        tokenizer.pad_token = tokenizer.eos_token
        model.config.pad_token_id = tokenizer.eos_token_id

    print(f"✅ Model loaded on GPU")
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

    # Training arguments optimized for 4090
    print("⚙️  Configuring training for RTX 4090...")
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
        # 4090 optimizations
        bf16=True,  # Use bfloat16 for speed
        fp16=False,
        save_safetensors=True,
        report_to="none",
        ddp_find_unused_parameters=False,
        # Memory optimizations
        max_grad_norm=1.0,
        gradient_checkpointing=True,
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
    print(f"  Device: CUDA (RTX 4090)")
    print(f"  Precision: BFloat16")
    print(f"  Learning rate: {LEARNING_RATE}")
    print()
    print("⏱️  Estimated time: 5-10 minutes on RTX 4090")
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
    model_save_path = "./ledit-security-4090"
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
    print("Next steps:")
    print("  1. Test the model: python3 -m transformers.pipeline text-generation ./ledit-security-4090")
    print("  2. Convert to GGUF for Ollama: python3 convert_to_gguf.py")
    print("  3. Run comprehensive tests: cd .. && go test ./pkg/security_validator/ -v")
    print()

if __name__ == "__main__":
    main()

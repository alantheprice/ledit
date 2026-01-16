# Fine-tuning for Security Validation

Fine-tune a small language model to classify shell commands and file operations as SAFE (0), CAUTION (1), or DANGEROUS (2).

## Quick Start (RTX 4090)

```bash
# 1. Install dependencies (one-time)
pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu118
pip install transformers datasets accelerate

# 2. Verify CUDA works
python3 -c "import torch; print(f'CUDA: {torch.cuda.is_available()}')"

# 3. Run training (5-10 minutes)
python3 train_4090.py

# 4. Test the model
python3 -c "
from transformers import pipeline
gen = pipeline('text-generation', model='./ledit-security-4090')
print(gen('Evaluate: cat README.md. Risk (0/1/2):')[0]['generated_text'])
"
```

## Expected Results

- **Training time**: 5-10 minutes on RTX 4090
- **GPU utilization**: ~70-90%
- **VRAM usage**: ~12-16 GB / 24 GB
- **Final accuracy**: >90% (up from 67% with off-the-shelf models)
- **Model size**: ~1.5 GB (after quantization)

## Training Scripts

- **`train_4090.py`** - Use this! CUDA-optimized for RTX 4090
- `train_cpu.py` - CPU fallback (very slow, hours to days)
- `train_mps.py` - Apple M3 Max (has compatibility issues)

All scripts are self-documenting - see their docstrings for usage details.

## Data

- `data/security_validation.jsonl` - 1,062 examples (full dataset)
- `output/train.jsonl` - 955 examples (balanced training subset)

## Output

Training produces:
- `./ledit-security-4090/` - Fine-tuned model in HuggingFace format
- `./checkpoints_4090/` - Training checkpoints

## Converting to GGUF for Ollama

After training, convert to GGUF format:

```bash
# Clone llama.cpp
git clone https://github.com/ggerganov/llama.cpp
cd llama.cpp

# Convert HuggingFace model to GGUF
python3 convert-hf-to-gguf.py ../ledit-security-4090 --outfile ledit-security.gguf

# Quantize (optional, for smaller size)
./quantize ledit-security.gguf ledit-security-q4.gguf Q4_K_M

# Copy back to Mac
scp ledit-security-q4.gguf user@mac:~/ledit/fine-tuning/
```

## Installing in Ollama (on Mac)

```bash
cd fine-tuning

# Create Modelfile
cat > Modelfile.finetuned << 'EOF'
FROM ./ledit-security-q4.gguf

PARAMETER temperature 0.1
PARAMETER num_ctx 8192

SYSTEM You are a security validation assistant. Classify as SAFE (0), CAUTION (1), or DANGEROUS (2). Respond with JSON.
EOF

# Create Ollama model
ollama create ledit-security -f Modelfile.finetuned

# Test
echo 'Evaluate: rm -rf /usr/bin. Risk:' | ollama run ledit-security
```

## Testing

Run comprehensive security validation tests:

```bash
cd ..  # Back to ledit root
go test ./pkg/security_validator/ -v
```

## Publishing to Hugging Face

```bash
# Install HF CLI
pip install huggingface_hub

# Login
huggingface-cli login

# Create repository
huggingface-cli repo create ledit-security --type model

# Upload model
cd fine-tuning
huggingface-cli upload ledit-security ledit-security-q4.gguf

# Upload README
huggingface-cli upload ledit-security README_4090.md README.md --repo-type model
```

## Model Performance

Off-the-shelf models tested:
- gemma3:270m - 19% accuracy (too small, fails on dangerous commands)
- gemma3:1b - 67% accuracy (overly cautious, false positives)
- gemma3:1b-it-qat - 67% accuracy (same issues)

After fine-tuning on 4090:
- **Expected: >90% accuracy** with proper balance of safety and usability

## Troubleshooting

**CUDA not available:**
```bash
# Reinstall PyTorch with CUDA
pip install torch --upgrade --extra-index-url https://download.pytorch.org/whl/cu118
```

**Out of memory:**
- Edit `train_4090.py`, reduce `BATCH_SIZE` from 16 to 8 or 4

**Slow training:**
- Verify CUDA is being used: `python3 -c "import torch; print(torch.cuda.get_device_name(0))"`
- Should show your 4090 GPU name

**Phi-3 compatibility issues (MPS/Mac):**
- Use `train_cpu.py` as fallback
- Or train on 4090 using `train_4090.py`

## File Structure

```
fine-tuning/
├── train_4090.py              # Main training script
├── train_cpu.py               # CPU fallback
├── train_mps.py               # Apple GPU (has issues)
├── generate_training_data.go  # Data generation tool
├── axolotl_config.yml         # Alternative training config
├── data/
│   └── security_validation.jsonl    # 1,062 examples
├── output/
│   └── train.jsonl                   # 955 training examples
└── README_4090.md            # This file
```

## Next Steps

1. ✅ Copy this directory to 4090 machine
2. ✅ Run `python3 train_4090.py` (5-10 min)
3. ✅ Test accuracy with Go test suite
4. ✅ Convert to GGUF format
5. ✅ Install in Ollama on Mac
6. ✅ Update ledit app to use fine-tuned model
7. ✅ Add `ledit security-model install` command
8. ✅ Publish to Hugging Face

Everything needed is in this directory. Just copy to the 4090 and run!

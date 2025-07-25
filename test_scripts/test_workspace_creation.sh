#!/bin/bash

# Function to return the test name
get_test_name() {
    echo "Initial Workspace Creation & Analysis"
}

# Function to run the test logic
run_test_logic() {
    local model_name=$1 # Capture the model_name passed from test.sh
    echo "--- TEST: Initial Workspace Creation & Analysis ---"
    # Create a diverse set of files
    echo "This is a text file." > file1.txt
    echo "print('hello from python')" > script.py
    echo '{"key": "value"}' > data.json

    # Create files and directories that should be ignored by the workspace scanner
    echo "log data" > ignored.log
    echo "secret" > .hidden_file
    echo "ignored.log" > .gitignore
    mkdir -p .hidden_dir
    echo "in hidden dir" > .hidden_dir/file.txt

    echo "--- Initial file structure created in 'testing/' directory: ---"
    ls -la
    echo "----------------------------------------------------------------"

    # Run ledit with #WORKSPACE to trigger the initial analysis.
    ../ledit code "Based on the files you see, create a 'manifest.txt' listing the text, python, and json files available. #WORKSPACE" --skip-prompt -m "$model_name"

    echo
    echo "--- Verifying Test ---"
    # Check that workspace.json was created
    if [ ! -f ".ledit/workspace.json" ]; then
        echo "FAIL: .ledit/workspace.json was not created."
        exit 1
    fi
    echo "PASS: .ledit/workspace.json was created."

    # Check that manifest.txt was created and has content
    if [ ! -s "manifest.txt" ]; then
        echo "FAIL: manifest.txt was not created or is empty."
        exit 1
    fi
    echo "PASS: manifest.txt was created and is not empty."
    echo "--- Content of manifest.txt: ---"
    cat manifest.txt
    echo "--------------------------------"

    # Check workspace.json for correct files
    grep -q "file1.txt" .ledit/workspace.json && echo "PASS: file1.txt found in workspace.json" || (echo "FAIL: file1.txt not in workspace.json"; exit 1)
    grep -q "script.py" .ledit/workspace.json && echo "PASS: script.py found in workspace.json" || (echo "FAIL: script.py not in workspace.json"; exit 1)
    grep -q "data.json" .ledit/workspace.json && echo "PASS: data.json found in workspace.json" || (echo "FAIL: data.json not in workspace.json"; exit 1)

    # Check that ignored/hidden files are NOT in workspace.json
    ! grep -q "ignored.log" .ledit/workspace.json && echo "PASS: ignored.log correctly ignored" || (echo "FAIL: ignored.log was found in workspace.json"; exit 1)
    ! grep -q ".hidden_file" .ledit/workspace.json && echo "PASS: .hidden_file correctly ignored" || (echo "FAIL: .hidden_file was found in workspace.json"; exit 1)
    ! grep -q ".hidden_dir" .ledit/workspace.json && echo "PASS: .hidden_dir correctly ignored" || (echo "FAIL: .hidden_dir was found in workspace.json"; exit 1)
    echo "----------------------------------------------------"
    echo
}
#!/usr/bin/env python3
"""Android App Implementation Script.

This script iterates through the Android app spec TODOs and helps execute them.
It reads SPEC.md and TODOS.md from each component folder in docs/android-app-spec/
and allows working through each item sequentially.

Usage:
    python3 android-app-impl/run_impl.py              # Interactive mode
    python3 android-app-impl/run_impl.py --list      # List all todos
    python3 android-app-impl/run_impl.py --component 01-go-mobile
    python3 android-app-impl/run_impl.py --complete "01-go-mobile: T001"
"""

import argparse
import os
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Optional

# ANSI colors
GREEN = '\033[32m'
YELLOW = '\033[33m'
BLUE = '\033[34m'
RED = '\033[31m'
RESET = '\033[0m'
BOLD = '\033[1m'

# Base paths
SCRIPT_DIR = Path(__file__).parent
SPEC_BASE = SCRIPT_DIR.parent / "docs" / "android-app-spec"

# Implementation order (folder -> name)
COMPONENT_ORDER = [
    ("01-go-mobile", "Go Mobile Bindings"),
    ("02-terminal-pty", "Terminal PTY Subsystem"),
    ("03-emulator-view", "Terminal Emulator View"),
    ("05-android-shell", "Android App Shell"),
    ("04-webui-integration", "WebUI Integration"),
    ("06-background-service", "Background Service"),
    ("07-shell-bundle", "Shell Bundle"),
]


@dataclass
class TodoItem:
    id: str
    description: str
    status: str  # pending, in_progress, completed
    completion_criteria: str
    priority: str = "medium"


@dataclass
class Component:
    folder: str
    name: str
    spec_path: Path
    todo_path: Path
    todos: list[TodoItem]


def parse_todos(todo_path: Path) -> list[TodoItem]:
    """Parse TODOS.md file and extract todo items."""
    todos = []
    if not todo_path.exists():
        return todos

    content = todo_path.read_text(encoding="utf-8")
    lines = content.splitlines()

    current_todo = None

    for line in lines:
        stripped = line.strip()
        
        # Match status markers like [ ] or [x] or [*]
        # Handle formats:
        # - "[ ] Description"
        # - "[x] Description"  
        # - "- [ ] **T001** Description"
        # - "- [ ] Description"
        status_match = re.match(r'^(-\s*)?\[([ x*])\]\s*(\*\*[A-Z]\d+\*\*\s*)?(.*)$', stripped)
        if status_match:
            # Optional leading "- "
            status_char = status_match.group(2)
            # Optional ID in **markdown**
            id_text = status_match.group(3) or ""
            text = status_match.group(4).strip().replace('**', '')

            # Determine status
            if status_char.lower() == 'x':
                status = "completed"
            elif status_char == '*':
                status = "in_progress"
            else:
                status = "pending"

            # Extract ID if present
            if id_text:
                id_clean = id_text.replace('*', '').strip()
                id_match = re.match(r'^([A-Z]\d+)$', id_clean)
                if id_match:
                    todo_id = id_match.group(1)
                else:
                    todo_id = f"T{len(todos) + 1:03d}"
            else:
                todo_id = f"T{len(todos) + 1:03d}"
            
            # Extract description (check for "T001 - Description" format)
            id_match = re.match(r'^([A-Z]\d+)\s*[-–—]\s*(.*)$', text)
            if id_match:
                todo_id = id_match.group(1)
                desc = id_match.group(2)
            else:
                desc = text

            current_todo = TodoItem(
                id=todo_id,
                description=desc,
                status=status,
                completion_criteria="",
                priority="medium"
            )
            todos.append(current_todo)
            continue
        
        # Handle "- **[ ]** Status: **pending**" format with "Todo:" and "Completion:" fields
        # This creates a placeholder that gets filled in by following lines
        dash_bold_match = re.match(r'^-\s+\*\*\[\s*\]\*\*\s+Status:\s+\*\*(\w+)\*\*', stripped)
        if dash_bold_match:
            status = dash_bold_match.group(1).lower()
            # Create new todo - description will be filled in by next "Todo:" line
            current_todo = TodoItem(
                id=f"T{len(todos) + 1:03d}",
                description="",
                status=status,
                completion_criteria="",
                priority="medium"
            )
            todos.append(current_todo)
            continue
        
        # Handle "Todo:" field - provides description for current todo
        todo_field_match = re.match(r'^-\s*Todo:\s*(.*)$', stripped)
        if todo_field_match and current_todo:
            current_todo.description = todo_field_match.group(1).strip()
            continue
        
        # Handle "Completion:" lines - add to current todo
        completion_match = re.match(r'^\*?Completion\*?:\s*(.*)$', stripped)
        if completion_match and current_todo:
            if not current_todo.completion_criteria:
                current_todo.completion_criteria = completion_match.group(1)
            else:
                current_todo.completion_criteria += " " + completion_match.group(1)
            continue
        
        # Handle "*Location*:" lines - could be useful metadata
        location_match = re.match(r'^\*Location\*:\s*(.*)$', stripped)
        if location_match and current_todo:
            current_todo.completion_criteria += f" (Location: {location_match.group(1)})"
            continue
        
        # Handle table row format: | Status | Todo Item | Completion Criteria |
        if stripped.startswith('|') and '---' not in stripped:
            parts = [p.strip() for p in stripped.split('|')]
            parts = [p for p in parts if p]
            
            if len(parts) >= 2:
                status_col = parts[0].lower()
                if 'pending' in status_col or 'completed' in status_col or 'in_progress' in status_col:
                    if 'completed' in status_col:
                        status = "completed"
                    elif 'in_progress' in status_col:
                        status = "in_progress"
                    else:
                        status = "pending"
                    
                    desc = parts[1] if len(parts) > 1 else ""
                    
                    id_match = re.match(r'^([A-Z]\d+)\s*[-–—]\s*(.*)$', desc)
                    if id_match:
                        todo_id = id_match.group(1)
                        desc = id_match.group(2)
                    else:
                        todo_id = f"T{len(todos) + 1:03d}"
                    
                    criteria = parts[2] if len(parts) > 2 else ""
                    
                    current_todo = TodoItem(
                        id=todo_id,
                        description=desc,
                        status=status,
                        completion_criteria=criteria,
                        priority="medium"
                    )
                    todos.append(current_todo)

    return todos


def load_components() -> list[Component]:
    """Load all components from the spec folders."""
    components = []

    for folder, name in COMPONENT_ORDER:
        spec_path = SPEC_BASE / folder / "SPEC.md"
        todo_path = SPEC_BASE / folder / "TODOS.md"

        if not spec_path.exists():
            print(f"{YELLOW}Warning: {spec_path} not found, skipping {folder}{RESET}")
            continue

        todos = parse_todos(todo_path)

        components.append(Component(
            folder=folder,
            name=name,
            spec_path=spec_path,
            todo_path=todo_path,
            todos=todos
        ))

    return components


def list_all_todos(components: list[Component]) -> None:
    """List all todos from all components."""
    print(f"\n{BOLD}=== All TODOs ==={RESET}\n")

    for comp in components:
        print(f"{BLUE}{comp.folder}: {comp.name}{RESET}")
        pending = [t for t in comp.todos if t.status == "pending"]
        in_progress = [t for t in comp.todos if t.status == "in_progress"]
        completed = [t for t in comp.todos if t.status == "completed"]

        if in_progress:
            print(f"  {YELLOW}In Progress:{RESET}")
            for t in in_progress:
                print(f"    {YELLOW}▸ {t.id}: {t.description}{RESET}")

        if pending:
            print(f"  {YELLOW}Pending:{RESET}")
            for t in pending:
                print(f"    ○ {t.id}: {t.description}")

        if completed:
            print(f"  {GREEN}Completed:{RESET}")
            for t in completed:
                print(f"    ✓ {t.id}: {t.description}")

        print()


def show_component_todos(comp: Component) -> None:
    """Show todos for a specific component."""
    print(f"\n{BOLD}=== {comp.folder}: {comp.name} ==={RESET}\n")

    if comp.todos:
        print(f"{BLUE}Spec:{RESET} {comp.spec_path}")
        print(f"{BLUE}Todo File:{RESET} {comp.todo_path}\n")
    else:
        print(f"{YELLOW}No TODOs found in {comp.todo_path}{RESET}")
        return

    pending = [t for t in comp.todos if t.status == "pending"]
    in_progress = [t for t in comp.todos if t.status == "in_progress"]
    completed = [t for t in comp.todos if t.status == "completed"]

    print(f"{BOLD}Pending ({len(pending)}):{RESET}")
    for t in pending:
        print(f"  ○ {t.id}: {t.description}")

    if in_progress:
        print(f"\n{YELLOW}In Progress ({len(in_progress)}):{RESET}")
        for t in in_progress:
            print(f"  ▸ {t.id}: {t.description}")

    if completed:
        print(f"\n{GREEN}Completed ({len(completed)}):{RESET}")
        for t in completed:
            print(f"  ✓ {t.id}: {t.description}")


def mark_todo_complete(todo_path: Path, todo_id: str) -> bool:
    """Mark a todo as completed in the TODOS.md file."""
    if not todo_path.exists():
        print(f"{RED}Error: {todo_path} not found{RESET}")
        return False

    content = todo_path.read_text(encoding="utf-8")

    # Find and replace the todo status
    # Match patterns like [ ] T001 - description or [ ] - description
    pattern = rf'^(\[{chr(32)}\])\s*({re.escape(todo_id)}\s*[-–—]\s*)(.*)$'

    new_content, count = re.subn(
        pattern,
        r'[\1]\2\3',
        content,
        flags=re.MULTILINE
    )

    # Try alternate pattern without ID
    if count == 0:
        # Try to find by ID anywhere in the line
        pattern2 = rf'^(\[{chr(32)}\])\s*.*{re.escape(todo_id)}.*[-–—]\s*(.*)$'
        new_content, count = re.subn(
            pattern2,
            r'[\1]\2',
            content,
            flags=re.MULTILINE
        )

    if count > 0:
        todo_path.write_text(new_content, encoding="utf-8")
        return True

    return False


def find_todo_by_id(components: list[Component], full_id: str) -> tuple[Optional[Component], Optional[TodoItem]]:
    """Find a todo by its full ID (e.g., '01-go-mobile: T001')."""
    parts = full_id.split(":")
    if len(parts) != 2:
        return None, None

    folder = parts[0].strip()
    todo_id = parts[1].strip()

    for comp in components:
        if comp.folder == folder:
            for todo in comp.todos:
                if todo.id == todo_id:
                    return comp, todo

    return None, None


def show_spec_summary(comp: Component) -> None:
    """Show a summary of the component spec."""
    if not comp.spec_path.exists():
        print(f"{RED}Spec not found: {comp.spec_path}{RESET}")
        return

    content = comp.spec_path.read_text(encoding="utf-8")

    # Extract key sections
    print(f"\n{BOLD}=== {comp.folder}: {comp.name} ==={RESET}\n")

    # Overview
    if "## Overview" in content:
        start = content.find("## Overview")
        end = content.find("##", start + 1)
        if end == -1:
            end = len(content)
        overview = content[start:end].strip()
        # Print first few paragraphs
        paras = overview.split("\n\n")
        for para in paras[1:4]:
            if para.strip():
                print(para[:500])
                print()


def interactive_mode(components: list[Component]) -> None:
    """Run in interactive mode."""
    print(f"\n{BOLD}Android App Implementation - Interactive Mode{RESET}\n")
    print(f"Base path: {SPEC_BASE}\n")

    while True:
        print(f"\n{BOLD}Available Components:{RESET}")
        for i, comp in enumerate(components):
            pending = len([t for t in comp.todos if t.status == "pending"])
            in_progress = len([t for t in comp.todos if t.status == "in_progress"])
            completed = len([t for t in comp.todos if t.status == "completed"])
            print(f"  {i + 1}. {comp.folder} - {comp.name} ({completed}/{len(comp.todos)} done, {pending} pending)")

        print("\nCommands:")
        print("  <number>  - Show component details")
        print("  s <num>   - Show component spec summary")
        print("  c <id>    - Mark todo complete (e.g., c 01-go-mobile: T001)")
        print("  l         - List all todos")
        print("  q         - Quit")

        try:
            cmd = input(f"\n{BOLD}> {RESET}").strip()
        except (EOFError, KeyboardInterrupt):
            print("\nExiting.")
            break

        if not cmd:
            continue

        if cmd.lower() == 'q':
            break
        elif cmd.lower() == 'l':
            list_all_todos(components)
        elif cmd.startswith('s '):
            # Show spec
            try:
                idx = int(cmd.split()[1]) - 1
                if 0 <= idx < len(components):
                    show_spec_summary(components[idx])
                else:
                    print(f"{RED}Invalid component number{RESET}")
            except (ValueError, IndexError):
                print(f"{RED}Invalid command{RESET}")
        elif cmd.startswith('c '):
            # Mark complete
            full_id = cmd[2:].strip()
            comp, todo = find_todo_by_id(components, full_id)
            if comp and todo:
                if mark_todo_complete(comp.todo_path, todo.id):
                    todo.status = "completed"
                    print(f"{GREEN}Marked {full_id} as completed{RESET}")
                else:
                    print(f"{RED}Failed to mark {full_id}{RESET}")
            else:
                print(f"{RED}Todo not found: {full_id}{RESET}")
        else:
            # Try as component number
            try:
                idx = int(cmd) - 1
                if 0 <= idx < len(components):
                    show_component_todos(components[idx])
                else:
                    print(f"{RED}Invalid component number{RESET}")
            except ValueError:
                print(f"{RED}Invalid command{RESET}")


def main():
    parser = argparse.ArgumentParser(
        description="Android App Implementation Script - Iterate through spec TODOs"
    )
    parser.add_argument(
        "--list", "-l",
        action="store_true",
        help="List all todos from all components"
    )
    parser.add_argument(
        "--component", "-c",
        type=str,
        help="Show todos for a specific component (e.g., 01-go-mobile)"
    )
    parser.add_argument(
        "--complete", "-m",
        type=str,
        help="Mark a todo as complete (e.g., '01-go-mobile: T001')"
    )
    parser.add_argument(
        "--spec", "-s",
        type=str,
        help="Show spec summary for a component"
    )

    args = parser.parse_args()

    # Load components
    components = load_components()

    if not components:
        print(f"{RED}No components found in {SPEC_BASE}{RESET}")
        sys.exit(1)

    if args.list:
        list_all_todos(components)
    elif args.component:
        for comp in components:
            if comp.folder == args.component:
                show_component_todos(comp)
                break
        else:
            print(f"{RED}Component not found: {args.component}{RESET}")
            sys.exit(1)
    elif args.spec:
        for comp in components:
            if comp.folder == args.spec:
                show_spec_summary(comp)
                break
        else:
            print(f"{RED}Component not found: {args.spec}{RESET}")
            sys.exit(1)
    elif args.complete:
        comp, todo = find_todo_by_id(components, args.complete)
        if comp and todo:
            if mark_todo_complete(comp.todo_path, todo.id):
                print(f"{GREEN}Marked {args.complete} as completed{RESET}")
            else:
                print(f"{RED}Failed to mark {args.complete}{RESET}")
                sys.exit(1)
        else:
            print(f"{RED}Todo not found: {args.complete}{RESET}")
            sys.exit(1)
    else:
        interactive_mode(components)


if __name__ == "__main__":
    main()
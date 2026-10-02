# PopScript

A statically typed, interpreted programming language designed for readability, explicit syntax, and deterministic parsing. Every line reads like a sentence.

```
lib import random
lib import random.number

list fruits = ["apple", "banana", "cherry"]

for fruit with fruits;
    int bonus = random.number(from=1, to=10)
    print("Got {fruit} with bonus {bonus}")
stop;
```

## Features

- **Statically typed** — all types declared explicitly: `int`, `float`, `string`, `bool`, `list`
- **Readable syntax** — `for fruit with fruits;` reads like plain English
- **String interpolation** — `print("Hello {name}!")`
- **GUI support** — built-in `ui` library powered by Fyne
- **Package manager** — `pop get package` downloads from Pop-Inc repository
- **Memory control** — optional manual memory management with `armemory=true`
- **if / elif / else** — full conditional support
- **when loops** — while-style loops with infinite loop protection
- **Functions with return** — `func`, `return`, proper scoping

## Quick Example

```
lib import ui

func on_click();
    string name = ui.get(name="input1")
    ui.set(name="label1", value="Hello {name}!")
stop;

ui.window(title="PopScript App", width=400, height=300)
ui.label(name="label1", text="Enter your name:")
ui.input(name="input1", placeholder="Name...")
ui.button(text="Greet", action="on_click")
ui.run()
```

## Installation

### Windows
Download `pop.exe` from [Releases](https://github.com/fallawerr-cmd/popscript/releases/latest) and add to PATH. (sorry win version is not ready)

### Linux
```bash
wget https://github.com/fallawerr-cmd/popscript/releases/latest/download/pop
chmod +x pop
sudo mv pop /usr/local/bin/
```

> Linux users also need: `sudo apt install libgl1-mesa-dev xorg-dev`

## Usage

```bash
pop run hello.pscript       # Run a script
pop get packagename         # Install a package
pop list                    # List installed packages
```

## Syntax Overview

```
$/ This is a comment

int x = 42
float pi = 3.14
string name = "PopScript"
bool flag = true
list nums = [1, 2, 3]

if x > 10;
    print("high")
elif x > 5;
    print("medium")
else;
    print("low")
stop;

int count = 0
when count < 5;
    print("count is {count}")
    int count = count + 1
stop;

for item with nums;
    print(item)
stop;

for i with num(10);
    print(i)
stop;

func add(a, b);
    return a + b
stop;

int result = add(3, 4)
print("Result: {result}")
```

## Built-in Libraries

| Library | Description |
|---------|-------------|
| `random` | Random numbers and letters |
| `ui` | Native GUI windows |
| `file` | File system operations |
| `time` | Date, time, sleep |

## Building from Source

```bash
git clone https://github.com/fallawerr-cmd/popscript
cd popscript
go mod init pop
go get fyne.io/fyne/v2
go mod tidy
go build -o pop
```

Requires Go 1.22+ and GCC (for Fyne/CGO).

## Project Structure

```
popscript/
  lexer/        — tokenizer
  ast/          — abstract syntax tree nodes
  parser/       — builds AST from tokens
  interpreter/  — executes AST
  ui/           — GUI library (Fyne wrapper)
  main.go       — CLI entry point (pop command)
```

## License

[MyOwnV1.0](LICENSE)

## Links

- Website: [popscr.github.io](https://popscr.github.io)
- Package repository: [github.com/fallawerr-cmd/plibs/tree/main](https://github.com/fallawerr-cmd/plibs/tree/main)
- Email: fallawerr@gmail.com

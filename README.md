# ⚡ HACKLAB

> Your terminal hacking playground.

Spin up vulnerable lab environments with one command, track objectives, and hack at your own pace — all from the terminal.

```
H   H  AAAAA  CCCC   K   K  L      AAAAA  BBBB
H   H  A   A  C      K  K   L      A   A  B   B
HHHHH  AAAAA  C      KKK    L      AAAAA  BBBB
H   H  A   A  C      K  K   L      A   A  B   B
H   H  A   A  CCCC   K   K  LLLLL  A   A  BBBB
```

## Requirements

- **Go 1.21+** (to build from source)
- **Docker** (to run lab containers)
- **Docker Compose** (usually bundled with Docker Desktop)

## Install

### From source

```bash
git clone https://github.com/HackLab-cli/hacklab.git
cd hacklab
go build -o hacklab .
sudo mv hacklab /usr/local/bin/
```

### Go install

```bash
go install github.com/HackLab-cli/hacklab@latest
```

### Verify

```bash
hacklab --version
hacklab --help
```

## Commands

| Command | Description |
|---------|-------------|
| `hacklab list` | List all installed labs with slug, name, difficulty, objective count |
| `hacklab start <slug>` | Spin up Docker containers and launch the interactive TUI |
| `hacklab status` | Show currently running labs |
| `hacklab stop <slug>` | Tear down a lab's containers |
| `hacklab add <source>` | Install a lab from a git repo, alias, or local path |
| `hacklab remove <slug>` | Remove a lab from your collection |

## Lab Aliases

HackLab includes shorthand aliases for popular lab collections:

| Alias | Repository | Example |
|-------|-----------|---------|
| `official` | [HackLab-cli/official-labs](https://github.com/HackLab-cli/official-labs) | `hacklab add official/dvwa` |
| `examples` | [HackLab-cli/lab-examples](https://github.com/HackLab-cli/lab-examples) | `hacklab add examples/juice-shop` |

```bash
# Add an official lab
hacklab add official/dvwa
hacklab add official/webgoat
hacklab add official/dvna

# Add an example lab
hacklab add examples/sqli-lab
hacklab add examples/juice-shop

# Or use full git URLs
hacklab add https://github.com/user/my-labs#path/to/lab
```

## Interactive TUI

Starting a lab opens a fullscreen terminal session. It's a **checklist / todo tracker** — no flag guessing, no quizzes. You hack, you check it off.

```
  ⚡ OWASP Juice Shop
  📡 http://localhost:3000

  2/5  ████████░░░░░░░░░░░░  40%

  ──────────────────────────────────────────────────────────

    ○  1. Bypass login with SQL injection        [injection]
    ○  2. Steal admin JWT token                  [auth]
  ▸ ○  3. Access admin panel                     [broken-auth]
    ○  4. Find forgotten backup file              [sensitive-data]
    ○  5. Exploit directory traversal             [injection]

  ──────────────────────────────────────────────────────────
  ↑/↓ navigate  ·  space/enter toggle  ·  h hint  ·  q quit
```

**Controls:**
- `↑/↓` or `j/k` — navigate objectives
- `space` / `enter` — mark an objective as done
- `h` — reveal hints for the selected objective
- `q` — quit

Progress saves automatically to `~/.hacklab/progress.json`.

## Starting guide

The first time you install `hacklab` without prior labs, it will guide you through the process and how to add some example labs! You can also browse official labs with `hacklab add official/<name>` — see the [official-labs repo](https://github.com/HackLab-cli/official-labs) for the full list.

## Lab Format

Labs are directories with a `lab.yml` manifest. Just create a folder, write the YAML, and you've got a lab.

### Single container

```yaml
name: OWASP Juice Shop
version: "1.0"
description: "Find and exploit 5 vulnerabilities in a modern web app"
difficulty: beginner
author: you
tags: [web, owasp, sqli, xss]

image: bkimminich/juice-shop:latest
port: 3000

wait_for: "http://localhost:3000"
wait_secs: 45

objectives:
  - name: "Bypass login with SQL injection"
    category: "injection"
    hints:
      - "Try the email field"
      - "Classic: ' OR 1=1 --"

  - name: "Perform a reflected XSS attack"
    category: "xss"
    hints:
      - "Look for search functionality"
      - "Try injecting a script tag"
```

### Multi-container (docker-compose)

```yaml
name: SQL Injection Lab
version: "1.0"
description: "Practice SQLi against PHP + MySQL"
difficulty: beginner

compose_file: docker-compose.yml

wait_for: "http://localhost:8080"
wait_secs: 30

objectives:
  - name: "Bypass login authentication"
    category: "sqli"
    hints:
      - "Try: admin' -- "

  - name: "Dump user credentials"
    category: "sqli"
    hints:
      - "Use UNION-based injection"
      - "Table is called 'users'"
```

Drop a `docker-compose.yml` in the same directory. Containers are automatically labelled with `hacklab.lab=<slug>` for tracking.

### Manifest fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name of the lab |
| `description` | No | Shown on the welcome screen |
| `difficulty` | No | beginner / intermediate / advanced |
| `author` | No | Your name/handle |
| `tags` | No | List of searchable keywords |
| `image` | One of* | Docker image to pull and run |
| `compose_file` | One of* | Path to docker-compose.yml |
| `port` | With `image` | Host port to bind |
| `wait_for` | No | URL to poll for readiness |
| `wait_secs` | No | Max seconds to wait (default: 30) |
| `objectives` | Yes | List of challenges |

### Objective fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | What the hacker needs to accomplish |
| `category` | No | Tag shown next to the objective |
| `hint` | No | Single hint string |
| `hints` | No | List of progressive hints |

## Storage

Everything is local, no cloud, no accounts:

```
~/.hacklab/
├── labs/              ← your installed labs
│   ├── juice-shop/
│   │   └── lab.yml
│   └── jwt-lab/
│       └── lab.yml
└── progress.json      ← completion state
```

## Why Hacklab?

| | TryHackMe / HTB | VulnHub | **Hacklab** |
|---|:---:|:---:|:---:|
| Terminal-native | ✗ | ✗ | ✓ |
| One-command setup | ✗ | ✗ | ✓ |
| No account needed | ✗ | ✓ | ✓ |
| Local-first | ✗ | ✓ | ✓ |
| Extensible via YAML | ✗ | ✗ | ✓ |
| Progress tracking | ✓ | ✗ | ✓ |
| Offline capable | ✗ | ✓ | ✓ |

## Extending

Labs are just folders with a `lab.yml`. That's it. No API to learn, no SDK. Write one, share it as a git repo, and anyone can install it with `hacklab add`.

## License

MIT

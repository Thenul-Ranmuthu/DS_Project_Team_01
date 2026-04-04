# DS_Project_Team_01
DS Project Repository (Team-01)

## Single-Source Multi-Node Setup (Best Practice)

Run all cluster instances from the single `node` codebase using different env files.

### Env presets

Inside `node/`:

- `.env.leader` -> port `5000`
- `.env.node1` -> port `5050`
- `.env.node2` -> port `5051`

Each file has a unique `NODE_ID`, `PORT`, and `PEERS` list.

### Start 3 nodes (Git Bash)

Open 3 terminals and run from `node/`:

```bash
ENV_FILE=.env.leader go run main.go
```

```bash
ENV_FILE=.env.node1 go run main.go
```

```bash
ENV_FILE=.env.node2 go run main.go
```

### Start 3 nodes (PowerShell)

From `node/` in three terminals:

```powershell
$env:ENV_FILE = '.env.leader'; go run main.go
```

```powershell
$env:ENV_FILE = '.env.node1'; go run main.go
```

```powershell
$env:ENV_FILE = '.env.node2'; go run main.go
```

### Why this setup

- One source of truth for code changes
- No drift between local cluster instances
- Easier commits, reviews, and CI/CD


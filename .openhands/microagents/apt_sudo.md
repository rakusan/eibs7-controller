---
name: apt-sudo
type: knowledge
version: 1.0.0
agent: CodeActAgent
triggers: []
---

# Apt Sudo Microagent

This microagent ensures that any usage of `apt-get` for installing packages is prefixed with `sudo`. It can be used in scripts or commands where elevated privileges are required.

**Usage Example:**
```
sudo apt-get update && sudo apt-get install -y <package>
```
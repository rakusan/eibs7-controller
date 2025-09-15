# Development Guidelines
Any usage of `apt-get` for installing packages is prefixed with `sudo`. It can be used in scripts or commands where elevated privileges are required. Example: `sudo apt-get update && sudo apt-get install -y <package>`

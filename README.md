# Bot Maze Trap

## Overview
The **Bot Maze Trap** is a sophisticated honeypot designed to **trap and slow down bots** by leading them into an endless maze of dynamically generated pages, all while gradually reducing their TCP window size and forcing large, pointless downloads.

This tool is particularly useful for **wasting the resources of malicious scrapers, spammers, and bots** that attempt to navigate a site in an automated fashion.

Friendly words file is borrowed from: https://github.com/glitchdotcom/friendly-words/tree/main


## Features
- **Infinite Maze Navigation:** Bots are redirected into an ever-expanding network of fake pages.
- **Progressive TCP Window Size Reduction:** Over multiple requests, bots experience progressively slower response times.
- **Forced Random Data Downloads:** 10MB GZIP-compressed random data files are served to bots.
- **External File Triggers:** Bots randomly trigger large file downloads from Hetzner’s speed test dataset.
- **Minimal Impact on Human Users:** The root (`/`) serves normal website files while everything else is trapped.

## How It Works
1. **Initial Redirect**: Any request outside the root `/` is redirected into the maze (`/m/...`).
2. **Dynamic Path Generation**: Each visit generates a new set of random paths, ensuring no end.
3. **TCP Window Shrinking**: As the same bot makes more requests, the TCP window size shrinks, throttling their bandwidth.
4. **Random Downloads**:
   - A random data file is served with a progressively decreasing TCP window.
   - External large files (100MB, 1GB, 10GB) are randomly triggered to overload bot connections.

## Installation
```sh
# Clone the repository
git clone https://github.com/your-repo/bot-maze-trap.git
cd bot-maze-trap

# Build the application
CGO_ENABLED=0 go build -o bot-maze-trap main.go

# Run the application
./bot-maze-trap
```

## Usage
Once running, the bot trap listens on `127.0.0.1:8282`.

- **Accessing `/`** serves normal website content.
- **Accessing anything else** redirects the request into the bot maze.
- **Analyzing bot activity**: Run the following command to check active bot traffic:
  ```sh
  journalctl -u bot-maze-trap.service -f
  ```

## Systemd Service File (Optional)
To run as a system service:
```ini
[Unit]
Description=Bot Maze Trap
After=network.target

[Service]
ExecStart=/path/to/bot-maze-trap
Restart=always
User=nobody
NoNewPrivileges=true
ProtectSystem=full
ProtectHome=true
MemoryDenyWriteExecute=true

[Install]
WantedBy=multi-user.target
```
```sh
# Enable and start the service
sudo systemctl daemon-reload
sudo systemctl enable bot-maze-trap
sudo systemctl start bot-maze-trap
```

## License
This project is released under the MIT License.

## Disclaimer
This tool is designed **for research and security purposes only**. Do not deploy it on production environments without understanding its impact.


# gomatrix

A Matrix digital rain screensaver written in Go, inspired by the classic [cmatrix](https://github.com/abishekvashok/cmatrix).

Note: The GIF is rendered in a lower frame rate and color depth for demonstration purposes. The actual application runs smoothly.

<img src="media/demo.gif" width="400">

## Install

```bash
go install github.com/frodi-karlsson/gomatrix@latest
```

Or build from source:

```bash
git clone https://github.com/frodi-karlsson/gomatrix.git
cd gomatrix
go build
```

## Usage

Run with default green color:

```bash
gomatrix
```

Use different color schemes:

```bash
gomatrix -color green
gomatrix -color blue
gomatrix -color red
```

Press any key to exit.
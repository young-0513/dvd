# DVD 📀 Video
A [bouncing DVD](https://www.youtube.com/watch?v=QOtuX0jL85Y) screen saver for your terminal. You can configure tmux to start this after a period of being idle for extra fun.





# Install

## Using Go on any OS
  - `go install github.com/integrii/dvd/cmd/dvd@latest`

## Using Homebrew on MacOS
  - `brew tap integrii/dvd https://github.com/integrii/dvd`
  - `brew install --HEAD integrii/dvd/dvd`


# tmux Screen Saver

Run `dvd` as a tmux screen saver by using tmux’s lock mechanism. Just add the following to your `~/.tmux.conf`:

```
set -g lock-after-time 300 # idle seconds before activating
set -g lock-command "dvd" 
```

## Optionally bind a key to start it on demand:

- `bind-key C-s lock-client`             # press Prefix + C-s to start the saver

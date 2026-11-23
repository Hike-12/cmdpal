package config

import (
	"slices"
	"strings"
)

const (
	Git     = "GIT"
	Docker  = "DOCKER"
	Network = "NETWORK & PORTS"
	Media   = "MEDIA & ARCHIVES"
	System  = "SYSTEM & DEV"
)

func Catalog() []Recipe {
	var all []Recipe
	add := func(category, name, safety, command, windows, summary string, flags ...string) {
		r := Recipe{Category: category, Name: name, Safety: safety, Command: command, Windows: windows, Summary: summary}
		target := &r.Flags
		for i := 0; i < len(flags); i += 2 {
			if flags[i] == "@windows" {
				target = &r.WindowsFlags
				continue
			}
			*target = append(*target, Flag{flags[i], flags[i+1]})
		}
		switch category {
		case Git, Docker:
			r.Tags = []string{strings.ToLower(category)}
		case System:
			r.Tags = []string{strings.Fields(command)[0]}
		}
		all = append(all, r)
	}
	add(Git, "Working tree status", Safe, "git status --short --branch", "", "Check the current branch and changed files.", "--short", "Use one line per file", "--branch", "Include branch and tracking status")
	add(Git, "Undo last commit (soft)", Caution, "git reset --soft HEAD~1", "", "Move HEAD back one commit and leave its changes staged.", "--soft", "Keep the index and working tree")
	add(Git, "Interactive rebase", Caution, "git rebase --interactive HEAD~{{count}}", "", "Open the editor to reorder, edit or squash recent commits.", "--interactive", "Edit the commit sequence before replaying it")
	add(Git, "Switch branch", Caution, "git switch -- {{branch}}", "", "Switch your working tree to a local branch.", "--", "End option parsing")
	add(Git, "Delete merged branches", Destructive, "git branch --merged --format='%(refname:short)' | while IFS= read -r branch; do case \"$branch\" in main|master|develop) ;; *) git branch -d -- \"$branch\" ;; esac; done", "git branch --merged --format='%(refname:short)' | Where-Object { $_ -notin 'main','master','develop' } | ForEach-Object { git branch -d -- $_ }", "Delete branches merged into HEAD. Keep main, master, develop and checked out branches.", "--merged", "List branches reachable from HEAD", "--format", "Print branch names only", "-d", "Refuse to delete unmerged or checked out branches", "--", "End option parsing")
	add(Git, "Pop latest stash", Caution, "git stash pop", "", "Apply the latest stash and drop it if the apply succeeds.")
	add(Git, "List stashes", Safe, "git stash list", "", "List saved stashes in newest-first order.")
	add(Git, "Push with lease", Destructive, "git push --force-with-lease origin {{branch}}", "", "Replace the remote branch only if it matches your local tracking ref.", "--force-with-lease", "Refuse the update if the expected remote tip changed")
	add(Git, "Start bisect", Caution, "git bisect start", "", "Start a session to find the first bad commit.")
	add(Docker, "Prune unused images", Destructive, "docker image prune -a", "", "Remove every image without an associated container. Docker asks first.", "-a", "Include unused tagged images")
	add(Docker, "Prune unused volumes", Destructive, "docker volume prune --all", "", "Remove unused Docker volumes and their stored data. Docker asks first.", "--all", "Include named and anonymous volumes")
	add(Docker, "Tail container logs", Safe, "docker logs --follow --tail 100 {{container}}", "", "Show the last 100 log lines and follow new output.", "--follow", "Keep streaming new output", "--tail", "Limit initial output to 100 lines")
	add(Docker, "Open container shell", Caution, "docker exec -it {{container}} sh", "", "Open an interactive shell in a running container.", "-i", "Keep stdin open", "-t", "Allocate a terminal")
	add(Docker, "Build and start Compose", Caution, "docker compose up --build", "", "Build and start the services defined in this project.", "--build", "Build images before starting containers")
	add(Docker, "Container resource usage", Safe, "docker stats", "", "Stream CPU, memory, network and disk usage by container.")
	add(Network, "Find listener on a port", Safe, "lsof -nP -iTCP:{{port}} -sTCP:LISTEN", "Get-NetTCPConnection -LocalPort {{port}} -State Listen", "Find the process listening on a local TCP port.", "-n / -P", "Show numeric host and port values", "-iTCP", "Select the TCP port", "-sTCP:LISTEN", "Select listeners", "@windows", "", "-LocalPort", "Select the local TCP port", "-State", "Select listeners")
	add(Network, "Stop listener on a port", Destructive, "lsof -t -iTCP:{{port}} -sTCP:LISTEN | while IFS= read -r pid; do kill -TERM \"$pid\"; done", "Get-NetTCPConnection -LocalPort {{port}} -State Listen | Select-Object -ExpandProperty OwningProcess -Unique | ForEach-Object { Stop-Process -Id $_ }", "Stop each process listening on the local TCP port.", "-t", "Print process IDs only", "-iTCP / -sTCP:LISTEN", "Select the TCP listeners", "-r", "Read backslashes literally", "-TERM", "Send the termination signal", "@windows", "", "-LocalPort / -State", "Select the local TCP listener", "-ExpandProperty / -Unique", "Read distinct process IDs", "-Id", "Stop the matching process")
	add(Network, "Show public IP", Safe, "curl -fsS https://api.ipify.org", "curl.exe -fsS https://api.ipify.org", "Ask ipify for the public address of this connection.", "-f", "Fail on HTTP errors", "-s", "Hide progress output", "-S", "Show errors")
	add(Network, "Check HTTP status", Safe, "curl -sSI -- {{url}}", "curl.exe -sSI -- {{url}}", "Make a HEAD request and print the status line and headers.", "-s", "Hide progress output", "-S", "Show errors", "-I", "Fetch response headers only", "--", "End option parsing")
	add(Media, "Compress video", Caution, "ffmpeg -n -i {{file}} -c:v libx264 -crf {{crf}} {{output}}", "", "Encode an H.264 video. A higher CRF makes a smaller file.", "-n", "Never overwrite an existing output", "-i", "Read the input file", "-c:v", "Use the H.264 video encoder", "-crf", "Set the quality target")
	add(Media, "Extract MP3 audio", Caution, "ffmpeg -n -i {{file}} -vn -c:a libmp3lame {{output}}", "", "Encode the input's audio as MP3.", "-n", "Never overwrite an existing output", "-i", "Read the input file", "-vn", "Drop video", "-c:a", "Use the MP3 audio encoder")
	add(Media, "Create tar.gz archive", Caution, "tar -czf {{output}} -- {{file}}", "", "Pack a file or directory into a gzip-compressed tar archive.", "-c", "Create an archive", "-z", "Compress with gzip", "-f", "Use the specified archive file", "--", "End option parsing")
	add(Media, "Extract tar.gz archive", Caution, "tar -xzkf {{file}}", "", "Unpack a gzip-compressed tar archive without replacing existing files.", "-x", "Extract the archive", "-z", "Decompress gzip", "-k", "Keep existing files", "-f", "Read the specified archive")
	add(System, "Largest folders", Safe, "du -sk ./*/ | sort -nr | head -n 15", "Get-ChildItem -Directory | ForEach-Object { [pscustomobject]@{Folder=$_.Name; Bytes=(Get-ChildItem $_.FullName -File -Recurse | Measure-Object -Property Length -Sum).Sum} } | Sort-Object Bytes -Descending | Select-Object -First 15", "List the 15 largest visible folders in this directory by size.", "-s / -k", "Summarize each folder in KiB", "-n / -r", "Sort by size in descending order", "-n 15", "Show 15 entries", "@windows", "", "-Directory / -File / -Recurse", "Measure files under each directory", "-Property / -Sum", "Total bytes", "-Descending / -First", "Keep the 15 largest")
	add(System, "Watch command (2s)", Caution, "while true; do clear; sh -c {{command}}; sleep 2; done", "while ($true) { Clear-Host; Invoke-Expression {{command}}; Start-Sleep -Seconds 2 }", "Repeat a command every two seconds. Press Ctrl+C to stop.", "-c", "Run the supplied command", "@windows", "", "-Seconds", "Set the delay")
	add(System, "Generate Ed25519 key", Caution, "ssh-keygen -t ed25519 -C {{message}}", "", "Create an SSH key. ssh-keygen prompts for the file and passphrase.", "-t", "Use the Ed25519 algorithm", "-C", "Set the key comment")
	add(System, "Stop process by PID", Destructive, "kill -TERM {{pid}}", "Stop-Process -Id {{pid}}", "Stop a local process. Check the PID before running.", "-TERM", "Send the termination signal", "@windows", "", "-Id", "Choose the process ID")
	add(System, "Run Go tests", Safe, "go test ./...", "", "Run tests in all Go packages under this directory.")
	add(System, "Run Node tests", Safe, "npm test", "", "Run the project's npm test script.")
	all[len(all)-1].Tags = []string{"node"}
	add(System, "Run Rust tests", Safe, "cargo test", "", "Run tests in the current Cargo project.")
	all[len(all)-1].Tags = []string{"rust"}
	add(System, "Run Python tests", Safe, "python -m pytest", "", "Run pytest with the current Python interpreter.", "-m", "Run the installed pytest module")
	add(System, "Run default Make target", Caution, "make", "", "Run the default target in the local Makefile.")
	return all
}

func Pinned(recipes []Recipe, badges, history []string) []int {
	type match struct{ index, score int }
	var matches []match
	projectPins := make(map[string]int)
	for index, recipe := range recipes {
		score := 0
		for _, badge := range badges {
			tag, _, _ := strings.Cut(badge, ":")
			if slices.Contains(recipe.Tags, tag) && projectPins[tag] < 2 {
				projectPins[tag]++
				score = 1
				break
			}
		}
		prefix, _, _ := strings.Cut(recipe.Command, "{{")
		prefix = strings.TrimSpace(prefix)
		for _, command := range history {
			if command == prefix || strings.HasPrefix(command, prefix+" ") ||
				(len(strings.Fields(command)) >= 2 && strings.HasPrefix(prefix, command+" ")) {
				score += 2
			}
		}
		if score > 0 {
			matches = append(matches, match{index, score})
		}
	}
	slices.SortStableFunc(matches, func(a, b match) int { return b.score - a.score })
	var indices []int
	for _, match := range matches[:min(8, len(matches))] {
		indices = append(indices, match.index)
	}
	return indices
}

# IDE Integration

This document describes how to integrate gitbak with various Integrated Development Environments (IDEs).

## Visual Studio Code

Add gitbak to VS Code Tasks:

1. Create or edit `.vscode/tasks.json`:
```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Start gitbak",
      "type": "shell",
      "command": "gitbak",
      "isBackground": true,
      "problemMatcher": []
    }
  ]
}
```

2. Run via `Terminal > Run Task... > Start gitbak`

For more details on VS Code tasks, see [VS Code Tasks Documentation](https://code.visualstudio.com/docs/editor/tasks).

## JetBrains IDEs (GoLand, IntelliJ, etc.)

Add gitbak as an External Tool:

1. Go to `Preferences/Settings > Tools > External Tools`
2. Click `+` and configure:
   - Name: `gitbak`
   - Program: `gitbak`
   - Working directory: `$ProjectFileDir$`
   - Advanced Options: Check "Asynchronous execution"
3. Run from `Tools > External Tools > gitbak`

For more details, see [JetBrains External Tools Documentation](https://www.jetbrains.com/help/idea/configuring-third-party-tools.html#local-ext-tools).

## Emacs

Add to your `.emacs` or `init.el`:

```elisp
(defun start-gitbak ()
  "Start gitbak in the current project."
  (interactive)
  (let ((default-directory (or (projectile-project-root) default-directory)))
    (start-process "gitbak" "*gitbak*" "gitbak")))

(global-set-key (kbd "C-c g b") 'start-gitbak)
```

This assumes you have projectile installed. If not, you can simplify to just use the current directory.

## Manual Integration

You can always run gitbak in a separate terminal window while using your IDE:

```bash
# In a separate terminal, navigate to your project
cd /path/to/your/project

# Start gitbak
gitbak
```

This approach works with any editor or IDE.
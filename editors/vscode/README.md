# Numscript support for Visual Studio Code

## Installation

The Numscript Language Server must be installed for this extension to work.

#### From vscode marketplace

*TBA*

#### Manually

* Clone the repository
* Install vsce: `npm install -g vsce`
* Build the extension: `make build`
* Install: In VS Code, open the Command Palette and run "Extensions: Install from VSIX..." and open the .vsix that was generated in the last step

## Features

* Syntax highlighting

## Roadmap

* Display errors
* Goto declaration
* Code actions
* Code snippets
* Autocomplete

## Extension Settings

* `numscript.server-path`: path to the language server

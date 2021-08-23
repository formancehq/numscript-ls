import * as path from 'path';
import * as vscode from 'vscode';

import * as lc from 'vscode-languageclient/node';

let dbg = vscode.window.createOutputChannel("NumscriptOut")

let client: lc.LanguageClient;

export async function activate(ctx: vscode.ExtensionContext) {
  let serverPath: string = vscode.workspace.getConfiguration("numscript").get("server-path")
  dbg.appendLine("Configured server path: " + serverPath)

  let run: lc.Executable = {
    command: serverPath,
    options: {},
  }

  // If the extension is launched in debug mode then the debug server options are used
  // Otherwise the run options are used
  let serverOptions: lc.ServerOptions = { run: run, debug: run };

  let clientOptions: lc.LanguageClientOptions = {
    documentSelector: [{ scheme: 'file', language: 'numscript' }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher('**/.num')
    }
  };

  // Create the language client and start the client.
  client = new lc.LanguageClient(
    'languageServerNumscript',
    'Numscript Language Server',
    serverOptions,
    clientOptions
  );

  // Start the client. This will also launch the server
  client.start();

  const restartHandler = () => {
    dbg.appendLine("Requested server restart")
    client.stop().then(() => {
      dbg.appendLine("Restarting")
      client.start()
    })
  }

  ctx.subscriptions.push(vscode.commands.registerCommand("numscript.restart-server", restartHandler))
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

import * as path from 'path';
import { workspace, ExtensionContext, window } from 'vscode';

import * as lc from 'vscode-languageclient/node';

let dbg = window.createOutputChannel("NumscriptOut")

let client: lc.LanguageClient;

export function activate(context: ExtensionContext) {
  let serverPath: string = workspace.getConfiguration("numscript").get("server-path")
  dbg.appendLine("Configured server path: " + serverPath)

  // The debug options for the server
  // --inspect=6009: runs the server in Node's Inspector mode so VS Code can attach to the server for debugging
  let debugOptions = { execArgv: ['--nolazy', '--inspect=6009'] };

  let run: lc.Executable = {
    command: serverPath,
    options: {},
  }

  // If the extension is launched in debug mode then the debug server options are used
  // Otherwise the run options are used
  let serverOptions: lc.ServerOptions = {
    run: run,
    debug: run
  };

  // Options to control the language client
  let clientOptions: lc.LanguageClientOptions = {
    // Register the server for plain text documents
    documentSelector: [{ scheme: 'file', language: 'numscript' }],
    synchronize: {
      // Notify the server about file changes to '.num files contained in the workspace
      fileEvents: workspace.createFileSystemWatcher('**/.num')
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
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

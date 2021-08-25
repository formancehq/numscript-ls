import * as path from 'path';
import * as vscode from 'vscode';
import { fetch } from 'node-fetch';

import * as lc from 'vscode-languageclient/node';

let dbg = vscode.window.createOutputChannel("Numscript Extension Output");

let client: lc.LanguageClient;

export async function fetchReleaseInfo(): Promise<GithubRelease> {
  const requestUrl = "api.github.com/repos/numary/numscript-ls/releases/latest";

  const response = await fetch(
      requestUrl,
      {
        headers: { Accept: "application/vnd.github.v3+json" }
      }
    );

  if (!response.ok) {
      dbg.appendLine("Error fetching latest release info");

      throw new Error(
          `Got response ${response.status} when trying to fetch latest release`
      );
  }

  return await response.json();
}

export interface GithubRelease {
  name: string;
  id: number;
  published_at: string;
  assets: Array<{
      name: string;
      browser_download_url: vscode.Uri;
  }>;
}

async function downloadServer() {
  let platforms = {
    "x64 linux": "linux-x64",
    "x64 darwin": "macos-x64",
    "arm64 darwin": "macos-x64",
  }

  let platform = platforms[`${process.arch} ${process.platform}`]
  if (platform === "undefined") {
    await vscode.window.showErrorMessage(
      "Your platform does not have prebuilt language server binaries yet, " +
      "you have to clone numary/numscript-ls and build the server yourself, " +
      "then set the server path in the Numscript vscode extension's settings." 
    )
  }
}

async function resolveServerPath(ctx: vscode.ExtensionContext): Promise<string> {
  let serverPath: string = vscode.workspace.getConfiguration("numscript").get("server-path");
  dbg.appendLine("Configured server path: " + serverPath);

  if (serverPath == null) {
    vscode.workspace.fs.createDirectory(ctx.globalStorageUri);
    let globalStorage = path.parse(ctx.globalStorageUri.fsPath);
    let releaseInfo = await fetchReleaseInfo()

    let currentServerTimestamp = ctx.globalState["serverTimestamp"]
    if (currentServerTimestamp !== releaseInfo.published_at) {
      vscode.window.showInformationMessage("Do you want to download the language server ?", "Yes", "No")
      downloadServer()
    }
  }

  return serverPath;
}

export async function activate(ctx: vscode.ExtensionContext) {
  

  let serverPath = await resolveServerPath(ctx);

  let run: lc.Executable = {
    command: serverPath,
    options: {},
  };

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
  };

  ctx.subscriptions.push(vscode.commands.registerCommand("numscript.restart-server", restartHandler));
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

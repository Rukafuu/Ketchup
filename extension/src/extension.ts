import * as vscode from 'vscode';
import { exec } from 'child_process';
import { promisify } from 'util';
import * as path from 'path';

const execAsync = promisify(exec);

interface ProviderStatus {
    name: string;
    health: 'clean' | 'drifted' | 'unknown';
    summary: string;
    findings: Finding[];
}

interface Finding {
    code: string;
    severity: string;
    summary: string;
    details: { key: string; value: string }[];
}

class FastForwardTreeItem extends vscode.TreeItem {
    constructor(
        public readonly label: string,
        public readonly collapsibleState: vscode.TreeItemCollapsibleState,
        public readonly contextValue?: string,
        public readonly iconPath?: vscode.ThemeIcon
    ) {
        super(label, collapsibleState);
    }
}

class FastForwardProvider implements vscode.TreeDataProvider<vscode.TreeItem> {
    private _onDidChangeTreeData: vscode.EventEmitter<vscode.TreeItem | undefined | null | void> = new vscode.EventEmitter();
    readonly onDidChangeTreeData: vscode.Event<vscode.TreeItem | undefined | null | void> = this._onDidChangeTreeData.event;

    private cliPath: string = 'ff';
    private workspaceRoot: string | undefined;
    private statusItems: ProviderStatus[] = [];

    constructor(private context: vscode.ExtensionContext) {
        this.updateConfiguration();
        
        if (vscode.workspace.workspaceFolders && vscode.workspace.workspaceFolders.length > 0) {
            this.workspaceRoot = vscode.workspace.workspaceFolders[0].uri.fsPath;
        }
    }

    private updateConfiguration() {
        const config = vscode.workspace.getConfiguration('fastforward');
        this.cliPath = config.get<string>('cliPath') || 'ff';
    }

    refresh(): void {
        this.updateConfiguration();
        this._onDidChangeTreeData.fire();
    }

    getTreeItem(element: vscode.TreeItem): vscode.TreeItem {
        return element;
    }

    async getChildren(element?: vscode.TreeItem): Promise<vscode.TreeItem[]> {
        if (!this.workspaceRoot) {
            return [new FastForwardTreeItem('No workspace open', vscode.TreeItemCollapsibleState.None)];
        }

        if (!element) {
            // Root level - show providers
            await this.fetchStatus();
            
            if (this.statusItems.length === 0) {
                return [new FastForwardTreeItem('Click refresh to check status', vscode.TreeItemCollapsibleState.None)];
            }

            return this.statusItems.map(status => {
                const icon = this.getHealthIcon(status.health);
                const item = new FastForwardTreeItem(
                    `${status.name}: ${status.summary}`,
                    status.findings.length > 0 
                        ? vscode.TreeItemCollapsibleState.Expanded 
                        : vscode.TreeItemCollapsibleState.None,
                    'provider',
                    icon
                );
                item.tooltip = `${status.name}\nHealth: ${status.health}\n${status.summary}`;
                item.command = { command: 'fastforward.diff', title: 'Show Diff' };
                return item;
            });
        }

        // This would be for findings under a provider, but we're simplifying
        return [];
    }

    private getHealthIcon(health: string): vscode.ThemeIcon {
        switch (health) {
            case 'clean': return new vscode.ThemeIcon('check', new vscode.ThemeColor('terminal.ansiGreen'));
            case 'drifted': return new vscode.ThemeIcon('warning', new vscode.ThemeColor('terminal.ansiYellow'));
            default: return new vscode.ThemeIcon('question', new vscode.ThemeColor('terminal.ansiBlue'));
        }
    }

    private async fetchStatus(): Promise<void> {
        try {
            const { stdout } = await execAsync(`${this.cliPath} status --json`, {
                cwd: this.workspaceRoot!,
                env: process.env
            });

            this.statusItems = JSON.parse(stdout);
        } catch (error: any) {
            // If command fails or returns non-zero, try to parse what we can
            if (error.stdout) {
                try {
                    this.statusItems = JSON.parse(error.stdout);
                    return;
                } catch {}
            }
            
            // Fallback: run without --json and parse manually
            try {
                const { stdout: textOutput } = await execAsync(`${this.cliPath} status`, {
                    cwd: this.workspaceRoot!,
                    env: process.env
                });
                
                // Simple parsing of text output
                this.statusItems = this.parseTextOutput(textOutput);
            } catch (parseError) {
                console.error('FastForward status error:', error);
                this.statusItems = [];
            }
        }
    }

    private parseTextOutput(output: string): ProviderStatus[] {
        const items: ProviderStatus[] = [];
        const lines = output.split('\n');
        
        let currentItem: Partial<ProviderStatus> = {};
        
        for (const line of lines) {
            const match = line.match(/^\[(.)\]\s+(\w+):\s+(.+)$/);
            if (match) {
                if (currentItem.name) {
                    items.push(currentItem as ProviderStatus);
                }
                
                const [, icon, name, summary] = match;
                currentItem = {
                    name,
                    summary: summary.trim(),
                    health: icon === '✓' ? 'clean' : icon === '!' ? 'drifted' : 'unknown',
                    findings: []
                };
            } else if (currentItem.name && line.trim().startsWith('•')) {
                currentItem.findings = currentItem.findings || [];
                currentItem.findings.push({
                    code: '',
                    severity: '',
                    summary: line.trim().substring(1).trim(),
                    details: []
                });
            }
        }
        
        if (currentItem.name) {
            items.push(currentItem as ProviderStatus);
        }
        
        return items;
    }
}

export function activate(context: vscode.ExtensionContext) {
    console.log('FastForward extension is now active');

    const provider = new FastForwardProvider(context);
    
    // Register tree view
    const treeView = vscode.window.createTreeView('fastforwardView', {
        treeDataProvider: provider,
        showCollapseAll: true
    });

    // Register commands
    context.subscriptions.push(
        vscode.commands.registerCommand('fastforward.refresh', () => {
            provider.refresh();
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('fastforward.status', async () => {
            await runFFCommand(context, 'status', true);
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('fastforward.diff', async () => {
            await runFFCommand(context, 'diff', true);
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('fastforward.sync', async () => {
            await runFFCommand(context, 'sync', false);
        })
    );

    context.subscriptions.push(
        vscode.commands.registerCommand('fastforward.doctor', async () => {
            await runFFCommand(context, 'doctor', true);
        })
    );

    // Auto-check on workspace open
    const config = vscode.workspace.getConfiguration('fastforward');
    if (config.get<boolean>('autoCheckOnOpen', true)) {
        provider.refresh();
    }

    // Listen for configuration changes
    context.subscriptions.push(
        vscode.workspace.onDidChangeConfiguration(e => {
            if (e.affectsConfiguration('fastforward')) {
                provider.refresh();
            }
        })
    );
}

async function runFFCommand(context: vscode.ExtensionContext, command: string, showOutput: boolean) {
    const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
    if (!workspaceRoot) {
        vscode.window.showErrorMessage('No workspace folder open');
        return;
    }

    const config = vscode.workspace.getConfiguration('fastforward');
    const cliPath = config.get<string>('cliPath') || 'ff';
    const showNotifications = config.get<boolean>('showNotifications', true);

    const outputChannel = vscode.window.createOutputChannel('FastForward');
    outputChannel.appendLine(`Running: ${cliPath} ${command}`);
    outputChannel.appendLine(`Working directory: ${workspaceRoot}`);
    outputChannel.appendLine('---');

    try {
        const { stdout, stderr } = await execAsync(`${cliPath} ${command}`, {
            cwd: workspaceRoot,
            env: process.env,
            maxBuffer: 1024 * 1024 // 1MB buffer
        });

        outputChannel.appendLine(stdout);
        if (stderr) {
            outputChannel.appendLine(stderr);
        }
        outputChannel.appendLine('---');
        outputChannel.appendLine(`Command completed with exit code 0`);

        if (showOutput) {
            outputChannel.show(true);
        }

        // Show notification for sync completion
        if (command === 'sync' && showNotifications) {
            if (stdout.includes('COMPLETED') || stdout.includes('already clean')) {
                vscode.window.showInformationMessage('FastForward sync completed successfully!');
            } else if (stdout.includes('MANUAL_REQUIRED')) {
                vscode.window.showWarningMessage('FastForward sync requires manual intervention.');
            }
        }

    } catch (error: any) {
        outputChannel.appendLine(`Error: ${error.message}`);
        if (error.stdout) {
            outputChannel.appendLine(error.stdout);
        }
        if (error.stderr) {
            outputChannel.appendLine(error.stderr);
        }
        outputChannel.show(true);

        const exitCode = error.code || 1;
        
        if (showNotifications) {
            if (exitCode === 1) {
                vscode.window.showWarningMessage('FastForward: Drift detected or action required');
            } else if (exitCode === 2) {
                vscode.window.showErrorMessage('FastForward: Configuration error');
            } else if (exitCode === 3) {
                vscode.window.showErrorMessage('FastForward: Check failed');
            } else {
                vscode.window.showErrorMessage(`FastForward error: ${error.message}`);
            }
        }
    }
}

export function deactivate() {}

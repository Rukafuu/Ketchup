"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.activate = activate;
exports.deactivate = deactivate;
const vscode = __importStar(require("vscode"));
const child_process_1 = require("child_process");
const util_1 = require("util");
const execAsync = (0, util_1.promisify)(child_process_1.exec);
class KetchupTreeItem extends vscode.TreeItem {
    constructor(label, collapsibleState, contextValue, iconPath, description) {
        super(label, collapsibleState);
        this.label = label;
        this.collapsibleState = collapsibleState;
        this.contextValue = contextValue;
        this.iconPath = iconPath;
        this.description = description;
    }
}
class KetchupProvider {
    constructor(context) {
        this.context = context;
        this._onDidChangeTreeData = new vscode.EventEmitter();
        this.onDidChangeTreeData = this._onDidChangeTreeData.event;
        this.cliPath = 'ketchup';
        this.statusItems = [];
        this.coreVersion = null;
        this.updateAvailable = false;
        this.statusBarItem = null;
        this.updateConfiguration();
        if (vscode.workspace.workspaceFolders && vscode.workspace.workspaceFolders.length > 0) {
            this.workspaceRoot = vscode.workspace.workspaceFolders[0].uri.fsPath;
        }
    }
    updateConfiguration() {
        const config = vscode.workspace.getConfiguration('ketchup');
        this.cliPath = config.get('cliPath') || 'ketchup';
        this.updateStatusBar();
    }
    refresh() {
        this.updateConfiguration();
        this._onDidChangeTreeData.fire();
    }
    getTreeItem(element) {
        return element;
    }
    async getChildren(element) {
        if (!this.workspaceRoot) {
            return [new KetchupTreeItem('No workspace open', vscode.TreeItemCollapsibleState.None)];
        }
        if (!element) {
            // Root level - show providers
            await this.fetchStatus();
            if (this.statusItems.length === 0) {
                return [new KetchupTreeItem('Click refresh to check status', vscode.TreeItemCollapsibleState.None)];
            }
            return this.statusItems.map(status => {
                const icon = this.getHealthIcon(status.health);
                const contextValue = status.name.toLowerCase().includes('git') ? 'git-provider' : 'provider';
                const item = new KetchupTreeItem(`${status.name}: ${status.summary}`, status.findings.length > 0
                    ? vscode.TreeItemCollapsibleState.Expanded
                    : vscode.TreeItemCollapsibleState.None, contextValue, icon, status.findings.length > 0 ? `${status.findings.length} issue(s)` : undefined);
                item.tooltip = `${status.name}\nHealth: ${status.health}\n${status.summary}`;
                item.command = { command: 'ketchup.diff', title: 'Show Diff' };
                return item;
            });
        }
        // Findings under a provider - quick-fix buttons
        const parentLabel = element?.label || '';
        const parentLabelStr = typeof parentLabel === 'string' ? parentLabel : '';
        const provider = this.statusItems.find(s => parentLabelStr.startsWith(s.name));
        if (provider && provider.findings.length > 0) {
            return provider.findings.map(finding => {
                const icon = this.getSeverityIcon(finding.severity);
                const item = new KetchupTreeItem(finding.summary, vscode.TreeItemCollapsibleState.None, 'finding', icon);
                item.tooltip = `${finding.code}\nSeverity: ${finding.severity}\n${finding.summary}`;
                // Add quick-fix command if available
                if (finding.code === 'GIT_DRIFT' || finding.code === 'GIT_BEHIND') {
                    item.command = { command: 'ketchup.catchup', title: 'Catch Up Branch' };
                }
                else if (finding.code.includes('ENV') || finding.code.includes('DEP')) {
                    item.command = { command: 'ketchup.sync', title: 'Sync Workspace' };
                }
                return item;
            });
        }
        return [];
    }
    getHealthIcon(health) {
        switch (health) {
            case 'clean': return new vscode.ThemeIcon('check', new vscode.ThemeColor('terminal.ansiGreen'));
            case 'drifted': return new vscode.ThemeIcon('warning', new vscode.ThemeColor('terminal.ansiYellow'));
            default: return new vscode.ThemeIcon('question', new vscode.ThemeColor('terminal.ansiBlue'));
        }
    }
    getSeverityIcon(severity) {
        if (severity.toLowerCase().includes('error') || severity.toLowerCase().includes('critical')) {
            return new vscode.ThemeIcon('error', new vscode.ThemeColor('terminal.ansiRed'));
        }
        else if (severity.toLowerCase().includes('warning')) {
            return new vscode.ThemeIcon('warning', new vscode.ThemeColor('terminal.ansiYellow'));
        }
        return new vscode.ThemeIcon('info', new vscode.ThemeColor('terminal.ansiBlue'));
    }
    async fetchStatus() {
        try {
            const { stdout } = await execAsync(`${this.cliPath} status --json`, {
                cwd: this.workspaceRoot,
                env: process.env
            });
            this.statusItems = JSON.parse(stdout);
            this.updateStatusBar();
        }
        catch (error) {
            // If command fails or returns non-zero, try to parse what we can
            if (error.stdout) {
                try {
                    this.statusItems = JSON.parse(error.stdout);
                    this.updateStatusBar();
                    return;
                }
                catch { }
            }
            // Fallback: run without --json and parse manually
            try {
                const { stdout: textOutput } = await execAsync(`${this.cliPath} status`, {
                    cwd: this.workspaceRoot,
                    env: process.env
                });
                // Simple parsing of text output
                this.statusItems = this.parseTextOutput(textOutput);
                this.updateStatusBar();
            }
            catch (parseError) {
                console.error('Ketchup status error:', error);
                this.statusItems = [];
            }
        }
    }
    async fetchCoreVersion() {
        try {
            const { stdout } = await execAsync(`${this.cliPath} version --json`, {
                cwd: this.workspaceRoot,
                env: process.env
            });
            this.coreVersion = JSON.parse(stdout);
            return this.coreVersion;
        }
        catch (error) {
            console.error('Failed to get core version:', error);
            return null;
        }
    }
    updateStatusBar() {
        const config = vscode.workspace.getConfiguration('ketchup');
        const showStatusBar = config.get('showStatusBar', true);
        if (!showStatusBar) {
            if (this.statusBarItem) {
                this.statusBarItem.dispose();
                this.statusBarItem = null;
            }
            return;
        }
        if (!this.statusBarItem) {
            this.statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 100);
            this.statusBarItem.command = 'ketchup.status';
            this.context.subscriptions.push(this.statusBarItem);
        }
        // Calculate overall health
        let hasDrift = false;
        let hasError = false;
        for (const status of this.statusItems) {
            if (status.health === 'drifted')
                hasDrift = true;
            if (status.health === 'unknown')
                hasError = true;
        }
        if (hasError) {
            this.statusBarItem.text = '$(error) Ketchup Error';
            this.statusBarItem.tooltip = 'Ketchup encountered an error checking status';
            this.statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.errorBackground');
        }
        else if (hasDrift) {
            this.statusBarItem.text = '$(warning) Ketchup Drift Detected';
            this.statusBarItem.tooltip = 'Ketchup detected drift in your workspace - click to view details';
            this.statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.warningBackground');
        }
        else if (this.statusItems.length > 0) {
            this.statusBarItem.text = '$(check) Ketchup Clean';
            this.statusBarItem.tooltip = 'Ketchup: All providers are clean';
            this.statusBarItem.backgroundColor = undefined;
        }
        else {
            this.statusBarItem.text = '$(git-pull-request) Ketchup';
            this.statusBarItem.tooltip = 'Ketchup: Click to check workspace status';
            this.statusBarItem.backgroundColor = undefined;
        }
        this.statusBarItem.show();
    }
    parseTextOutput(output) {
        const items = [];
        const lines = output.split('\n');
        let currentItem = {};
        for (const line of lines) {
            const match = line.match(/^\[(.)\]\s+(\w+):\s+(.+)$/);
            if (match) {
                if (currentItem.name) {
                    items.push(currentItem);
                }
                const [, icon, name, summary] = match;
                currentItem = {
                    name,
                    summary: summary.trim(),
                    health: icon === '✓' ? 'clean' : icon === '!' ? 'drifted' : 'unknown',
                    findings: []
                };
            }
            else if (currentItem.name && line.trim().startsWith('•')) {
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
            items.push(currentItem);
        }
        return items;
    }
}
function activate(context) {
    console.log('Ketchup extension is now active');
    const provider = new KetchupProvider(context);
    // Register tree view
    const treeView = vscode.window.createTreeView('ketchupView', {
        treeDataProvider: provider,
        showCollapseAll: true
    });
    // Register commands
    context.subscriptions.push(vscode.commands.registerCommand('ketchup.refresh', () => {
        provider.refresh();
    }));
    context.subscriptions.push(vscode.commands.registerCommand('ketchup.status', async () => {
        await runKetchupCommand(context, 'status', true);
    }));
    context.subscriptions.push(vscode.commands.registerCommand('ketchup.diff', async () => {
        await runKetchupCommand(context, 'diff', true);
    }));
    context.subscriptions.push(vscode.commands.registerCommand('ketchup.sync', async () => {
        await runKetchupCommand(context, 'sync', false);
    }));
    context.subscriptions.push(vscode.commands.registerCommand('ketchup.doctor', async () => {
        await runKetchupCommand(context, 'doctor', true);
    }));
    context.subscriptions.push(vscode.commands.registerCommand('ketchup.catchup', async () => {
        await runKetchupCommand(context, 'catch-up', false);
    }));
    context.subscriptions.push(vscode.commands.registerCommand('ketchup.update', async () => {
        await checkForUpdates(context, false);
    }));
    // Auto-check on workspace open
    const config = vscode.workspace.getConfiguration('ketchup');
    if (config.get('autoCheckOnOpen', true)) {
        provider.refresh();
    }
    // Auto-check for core updates on startup
    if (config.get('autoUpdate', true)) {
        checkForUpdates(context, true);
    }
    // Listen for configuration changes
    context.subscriptions.push(vscode.workspace.onDidChangeConfiguration(e => {
        if (e.affectsConfiguration('ketchup')) {
            provider.refresh();
        }
    }));
}
async function checkForUpdates(context, silent) {
    const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
    if (!workspaceRoot) {
        return;
    }
    const config = vscode.workspace.getConfiguration('ketchup');
    const cliPath = config.get('cliPath') || 'ketchup';
    const channel = config.get('updateChannel', 'stable');
    const outputChannel = vscode.window.createOutputChannel('Ketchup');
    outputChannel.appendLine(`Checking for Ketchup updates (${channel} channel)...`);
    try {
        const { stdout } = await execAsync(`${cliPath} update --check --channel ${channel}`, {
            cwd: workspaceRoot,
            env: process.env
        });
        outputChannel.appendLine(stdout);
        if (stdout.includes('update available') || stdout.includes('new version')) {
            if (!silent) {
                const action = await vscode.window.showInformationMessage('A new version of Ketchup core is available. Would you like to update?', 'Update Now', 'Later');
                if (action === 'Update Now') {
                    await runKetchupCommand(context, `update --channel ${channel}`, false);
                }
            }
            else {
                vscode.window.showInformationMessage('Ketchup update available - click the update button to install');
            }
        }
        else if (!silent) {
            vscode.window.showInformationMessage('Ketchup core is up to date');
        }
    }
    catch (error) {
        outputChannel.appendLine(`Error checking for updates: ${error.message}`);
        if (!silent) {
            if (error.message.includes('no update available') || error.message.includes('already up to date')) {
                vscode.window.showInformationMessage('Ketchup core is up to date');
            }
            else {
                vscode.window.showWarningMessage('Could not check for Ketchup updates. The update server may be unavailable.');
            }
        }
    }
}
async function runKetchupCommand(context, command, showOutput) {
    const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
    if (!workspaceRoot) {
        vscode.window.showErrorMessage('No workspace folder open');
        return;
    }
    const config = vscode.workspace.getConfiguration('ketchup');
    const cliPath = config.get('cliPath') || 'ketchup';
    const showNotifications = config.get('showNotifications', true);
    const outputChannel = vscode.window.createOutputChannel('Ketchup');
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
        // Show notification for sync/catchup completion
        if ((command === 'sync' || command === 'catch-up') && showNotifications) {
            if (stdout.includes('COMPLETED') || stdout.includes('already clean') || stdout.includes('up to date')) {
                vscode.window.showInformationMessage(`Ketchup ${command.split(' ')[0]} completed successfully!`);
            }
            else if (stdout.includes('MANUAL_REQUIRED')) {
                vscode.window.showWarningMessage(`Ketchup ${command.split(' ')[0]} requires manual intervention.`);
            }
        }
    }
    catch (error) {
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
                vscode.window.showWarningMessage('Ketchup: Drift detected or action required');
            }
            else if (exitCode === 2) {
                vscode.window.showErrorMessage('Ketchup: Configuration error');
            }
            else if (exitCode === 3) {
                vscode.window.showErrorMessage('Ketchup: Check failed');
            }
            else {
                vscode.window.showErrorMessage(`Ketchup error: ${error.message}`);
            }
        }
    }
}
function deactivate() { }
//# sourceMappingURL=extension.js.map
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
let outputChannel;
function normalizeHealth(health) {
    switch (health.toUpperCase()) {
        case 'CLEAN': return 'clean';
        case 'DRIFTED': return 'drifted';
        default: return 'unknown';
    }
}
function normalizeReports(raw) {
    return raw.map(report => ({
        name: report.provider || report.name || 'Unknown',
        health: normalizeHealth(report.health),
        summary: report.summary,
        findings: (report.findings || []).map(finding => ({
            code: finding.code,
            severity: String(finding.severity),
            summary: finding.summary,
            details: finding.details || []
        }))
    }));
}
class KetchupTreeItem extends vscode.TreeItem {
    constructor(label, collapsibleState, contextValue, iconPath, description, providerName) {
        super(label, collapsibleState);
        this.label = label;
        this.collapsibleState = collapsibleState;
        this.contextValue = contextValue;
        this.iconPath = iconPath;
        this.description = description;
        this.providerName = providerName;
    }
}
class KetchupProvider {
    constructor(context) {
        this.context = context;
        this._onDidChangeTreeData = new vscode.EventEmitter();
        this.onDidChangeTreeData = this._onDidChangeTreeData.event;
        this.cliPath = 'ketchup';
        this.statusItems = [];
        this.statusBarItem = null;
        this.lastError = null;
        this.updateConfiguration();
        if (vscode.workspace.workspaceFolders?.length) {
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
            await this.fetchStatus();
            if (this.lastError) {
                return [
                    new KetchupTreeItem(`Error: ${this.lastError}`, vscode.TreeItemCollapsibleState.None, undefined, new vscode.ThemeIcon('error', new vscode.ThemeColor('terminal.ansiRed'))),
                    new KetchupTreeItem('Run Doctor to diagnose setup', vscode.TreeItemCollapsibleState.None, undefined, new vscode.ThemeIcon('wrench'))
                ];
            }
            if (this.statusItems.length === 0) {
                return [new KetchupTreeItem('Click refresh to check status', vscode.TreeItemCollapsibleState.None)];
            }
            return this.statusItems.map(status => {
                const icon = this.getHealthIcon(status.health);
                const isGit = status.name.toLowerCase().includes('git');
                const contextValue = isGit ? 'git-provider' : 'provider';
                const item = new KetchupTreeItem(`${status.name}: ${status.summary}`, status.findings.length > 0
                    ? vscode.TreeItemCollapsibleState.Expanded
                    : vscode.TreeItemCollapsibleState.None, contextValue, icon, status.findings.length > 0 ? `${status.findings.length} issue(s)` : undefined, status.name);
                item.tooltip = `${status.name}\nHealth: ${status.health}\n${status.summary}`;
                item.command = { command: 'ketchup.diff', title: 'Show Diff' };
                return item;
            });
        }
        const treeItem = element;
        const provider = this.statusItems.find(s => s.name === treeItem.providerName);
        if (provider?.findings.length) {
            return provider.findings.map(finding => {
                const icon = this.getSeverityIcon(finding.severity);
                const item = new KetchupTreeItem(finding.summary, vscode.TreeItemCollapsibleState.None, 'finding', icon);
                item.tooltip = `${finding.code}\nSeverity: ${finding.severity}\n${finding.summary}`;
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
    getDriftCount() {
        return this.statusItems.filter(s => s.health === 'drifted').length;
    }
    getHealthIcon(health) {
        switch (health) {
            case 'clean': return new vscode.ThemeIcon('check', new vscode.ThemeColor('terminal.ansiGreen'));
            case 'drifted': return new vscode.ThemeIcon('warning', new vscode.ThemeColor('terminal.ansiYellow'));
            default: return new vscode.ThemeIcon('question', new vscode.ThemeColor('terminal.ansiBlue'));
        }
    }
    getSeverityIcon(severity) {
        const normalized = severity.toLowerCase();
        if (normalized.includes('error') || normalized.includes('critical')) {
            return new vscode.ThemeIcon('error', new vscode.ThemeColor('terminal.ansiRed'));
        }
        if (normalized.includes('warning')) {
            return new vscode.ThemeIcon('warning', new vscode.ThemeColor('terminal.ansiYellow'));
        }
        return new vscode.ThemeIcon('info', new vscode.ThemeColor('terminal.ansiBlue'));
    }
    async fetchStatus() {
        this.lastError = null;
        try {
            const { stdout } = await execAsync(`${this.cliPath} status --json`, {
                cwd: this.workspaceRoot,
                env: this.buildEnv()
            });
            this.statusItems = normalizeReports(JSON.parse(stdout));
            this.updateStatusBar();
            return;
        }
        catch (error) {
            const execError = error;
            if (execError.stdout) {
                try {
                    this.statusItems = normalizeReports(JSON.parse(execError.stdout));
                    this.updateStatusBar();
                    return;
                }
                catch {
                    // fall through to text parsing
                }
            }
        }
        try {
            const { stdout: textOutput } = await execAsync(`${this.cliPath} status`, {
                cwd: this.workspaceRoot,
                env: this.buildEnv()
            });
            this.statusItems = this.parseTextOutput(textOutput);
            this.updateStatusBar();
        }
        catch (error) {
            const execError = error;
            this.lastError = execError.message || 'Failed to run ketchup status';
            this.statusItems = [];
            this.updateStatusBar();
            console.error('Ketchup status error:', execError.stdout || execError.message);
        }
    }
    buildEnv() {
        const env = { ...process.env };
        const editor = vscode.window.activeTextEditor;
        if (editor) {
            env.KETCHUP_CURRENT_FILE = editor.document.uri.fsPath;
        }
        return env;
    }
    updateStatusBar() {
        const config = vscode.workspace.getConfiguration('ketchup');
        const showStatusBar = config.get('showStatusBar', true);
        if (!showStatusBar) {
            this.statusBarItem?.dispose();
            this.statusBarItem = null;
            return;
        }
        if (!this.statusBarItem) {
            this.statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 100);
            this.statusBarItem.command = 'ketchup.status';
            this.context.subscriptions.push(this.statusBarItem);
        }
        if (this.lastError) {
            this.statusBarItem.text = '$(error) Ketchup Error';
            this.statusBarItem.tooltip = this.lastError;
            this.statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.errorBackground');
        }
        else {
            const driftCount = this.statusItems.filter(s => s.health === 'drifted').length;
            const hasUnknown = this.statusItems.some(s => s.health === 'unknown');
            if (hasUnknown) {
                this.statusBarItem.text = '$(question) Ketchup Unknown';
                this.statusBarItem.tooltip = 'Some providers could not be checked';
                this.statusBarItem.backgroundColor = undefined;
            }
            else if (driftCount > 0) {
                this.statusBarItem.text = `$(warning) Ketchup (${driftCount})`;
                this.statusBarItem.tooltip = `${driftCount} provider(s) with drift — click for details`;
                this.statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.warningBackground');
            }
            else if (this.statusItems.length > 0) {
                this.statusBarItem.text = '$(check) Ketchup Clean';
                this.statusBarItem.tooltip = 'All providers are clean';
                this.statusBarItem.backgroundColor = undefined;
            }
            else {
                this.statusBarItem.text = '$(git-pull-request) Ketchup';
                this.statusBarItem.tooltip = 'Click to check workspace status';
                this.statusBarItem.backgroundColor = undefined;
            }
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
    outputChannel = vscode.window.createOutputChannel('Ketchup');
    context.subscriptions.push(outputChannel);
    const provider = new KetchupProvider(context);
    const treeView = vscode.window.createTreeView('ketchupView', {
        treeDataProvider: provider,
        showCollapseAll: true
    });
    const updateBadge = () => {
        const driftCount = provider.getDriftCount();
        treeView.badge = driftCount > 0 ? { value: driftCount, tooltip: `${driftCount} provider(s) with drift` } : undefined;
    };
    const refreshAll = () => {
        provider.refresh();
        updateBadge();
    };
    context.subscriptions.push(vscode.commands.registerCommand('ketchup.refresh', refreshAll), vscode.commands.registerCommand('ketchup.status', async () => {
        await runKetchupCommand(context, 'status', true);
        refreshAll();
    }), vscode.commands.registerCommand('ketchup.diff', async () => {
        await runKetchupCommand(context, 'diff', true);
    }), vscode.commands.registerCommand('ketchup.sync', async () => {
        await runKetchupCommand(context, 'sync', false);
        refreshAll();
    }), vscode.commands.registerCommand('ketchup.doctor', async () => {
        await runKetchupCommand(context, 'doctor', true);
    }), vscode.commands.registerCommand('ketchup.catchup', async () => {
        await runCatchUp(context, false);
        refreshAll();
    }), vscode.commands.registerCommand('ketchup.catchup.all', async () => {
        await runCatchUp(context, true);
        refreshAll();
    }), vscode.commands.registerCommand('ketchup.update', async () => {
        await checkForUpdates(context, false);
    }));
    const config = vscode.workspace.getConfiguration('ketchup');
    if (config.get('autoCheckOnOpen', true)) {
        refreshAll();
    }
    if (config.get('autoUpdate', true)) {
        checkForUpdates(context, true);
    }
    context.subscriptions.push(vscode.workspace.onDidChangeConfiguration(e => {
        if (e.affectsConfiguration('ketchup')) {
            refreshAll();
        }
    }));
}
function buildCatchUpCommand(forceAll = false) {
    const config = vscode.workspace.getConfiguration('ketchup');
    const show = forceAll ? 'all' : (config.get('catchUpShow', 'relevant') || 'relevant');
    const explain = config.get('catchUpExplain', false);
    const parts = ['catch-up'];
    if (show === 'all') {
        parts.push('--show', 'all');
    }
    if (explain) {
        parts.push('--explain');
    }
    return parts.join(' ');
}
async function runCatchUp(context, showAll) {
    await runKetchupCommand(context, buildCatchUpCommand(showAll), true);
}
async function checkForUpdates(context, silent) {
    const workspaceRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
    if (!workspaceRoot) {
        return;
    }
    const config = vscode.workspace.getConfiguration('ketchup');
    const cliPath = config.get('cliPath') || 'ketchup';
    const channel = config.get('updateChannel', 'stable');
    outputChannel.appendLine(`Checking for Ketchup updates (${channel} channel)...`);
    try {
        const { stdout } = await execAsync(`${cliPath} update --check --channel ${channel}`, {
            cwd: workspaceRoot,
            env: process.env
        });
        outputChannel.appendLine(stdout);
        if (stdout.toLowerCase().includes('update available')) {
            if (!silent) {
                const action = await vscode.window.showInformationMessage('A new version of Ketchup core is available. Would you like to update?', 'Update Now', 'Later');
                if (action === 'Update Now') {
                    await runKetchupCommand(context, `update --channel ${channel}`, false);
                }
            }
            else {
                vscode.window.showInformationMessage('Ketchup update available — use "Check for Updates" in the sidebar');
            }
        }
        else if (!silent) {
            vscode.window.showInformationMessage('Ketchup core is up to date');
        }
    }
    catch (error) {
        const execError = error;
        outputChannel.appendLine(`Error checking for updates: ${execError.message}`);
        if (!silent) {
            const message = execError.message?.toLowerCase() || '';
            if (message.includes('already on latest') || message.includes('no updates')) {
                vscode.window.showInformationMessage('Ketchup core is up to date');
            }
            else {
                vscode.window.showWarningMessage('Could not check for Ketchup updates. The update server may be unavailable.');
            }
        }
    }
}
function buildCommandEnv() {
    const env = { ...process.env };
    const editor = vscode.window.activeTextEditor;
    if (editor) {
        env.KETCHUP_CURRENT_FILE = editor.document.uri.fsPath;
    }
    return env;
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
    outputChannel.appendLine(`Running: ${cliPath} ${command}`);
    outputChannel.appendLine(`Working directory: ${workspaceRoot}`);
    outputChannel.appendLine('---');
    try {
        const { stdout, stderr } = await execAsync(`${cliPath} ${command}`, {
            cwd: workspaceRoot,
            env: buildCommandEnv(),
            maxBuffer: 1024 * 1024
        });
        outputChannel.appendLine(stdout);
        if (stderr) {
            outputChannel.appendLine(stderr);
        }
        outputChannel.appendLine('---');
        outputChannel.appendLine('Command completed with exit code 0');
        if (showOutput) {
            outputChannel.show(true);
        }
        const baseCommand = command.split(' ')[0];
        if ((baseCommand === 'sync' || baseCommand === 'catch-up') && showNotifications) {
            if (stdout.includes('COMPLETED') || stdout.includes('already clean') || stdout.includes('up to date')) {
                vscode.window.showInformationMessage(`Ketchup ${baseCommand} completed successfully`);
            }
            else if (stdout.includes('MANUAL_REQUIRED')) {
                vscode.window.showWarningMessage(`Ketchup ${baseCommand} requires manual intervention`);
            }
        }
    }
    catch (error) {
        const execError = error;
        outputChannel.appendLine(`Error: ${execError.message}`);
        if (execError.stdout) {
            outputChannel.appendLine(execError.stdout);
        }
        if (execError.stderr) {
            outputChannel.appendLine(execError.stderr);
        }
        outputChannel.show(true);
        const exitCode = execError.code || 1;
        if (showNotifications) {
            if (exitCode === 1) {
                vscode.window.showWarningMessage('Ketchup: drift detected or action required');
            }
            else if (exitCode === 2) {
                vscode.window.showErrorMessage('Ketchup: configuration error — run Doctor');
            }
            else if (exitCode === 3) {
                vscode.window.showErrorMessage('Ketchup: check failed');
            }
            else {
                vscode.window.showErrorMessage(`Ketchup error: ${execError.message}`);
            }
        }
    }
}
function deactivate() { }
//# sourceMappingURL=extension.js.map
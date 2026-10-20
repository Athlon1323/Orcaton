import { SvelteSet } from 'svelte/reactivity';
import { loadBackend, type Backend } from './backend';
import { dialogTexts, has, language, setLanguage, t } from './i18n';
import { newProgress, step, type Progress } from './progress';
import { baseName, bytes, dirName } from './format';
import type {
  AboutInfo, CLITool, ErrorInfo, FFmpegPlan, FileResult, FileResultKind, MissingFile, Options, ProjectInfo,
  ProjectReply, SearchReport, StartupInfo, Summary, SyncEvent,
} from './types';

export type Nav = 'import' | 'settings' | 'about' | 'dev';
export type Phase = 'empty' | 'ready' | 'syncing' | 'cancelled' | 'done';
export type View = 'timeline' | 'files';

export interface Result {
  summary: Summary;
  files: FileResult[];
}

export interface ToastAction {
  label: string;
  run: () => void;
}

export interface Toast {
  id: number;
  level: 'ok' | 'info' | 'warn' | 'err';
  title: string;
  text?: string;
  details?: string;
  actions?: ToastAction[];
}

export interface MissingAsk {
  files: MissingFile[];
  note: string;
}

const FALLBACK_OPTIONS: Options = {
  unsynced: 'order_time', orderBy: 'name', singles: 'sequential', unsyncedLabel: 'Rose', timeLabel: 'Mango', minConfidence: 12, minRatio: 1.5,
  autoThreshold: true, minOverlapSec: 2, maxOffsetSec: 0, audioMode: 'mix', fileOrder: true,
  intraLane: false, recordTime: true, maxGapSec: 0, keepGapSec: 0, jobs: 0, threads: 0, memLimitMB: 0, cacheDir: '',
};

const LOG_MAX = 1000;

export function failed(e: unknown): ErrorInfo {
  return { kind: 'failed', message: e instanceof Error ? e.message : String(e) };
}

export class AppState {
  backend: Backend | null = null;
  ready = $state(false);
  outside = $state(false);
  startup = $state<StartupInfo | null>(null);
  nav = $state<Nav>('import');
  phase = $state<Phase>('empty');
  view = $state<View>('files');
  ba = $state<'before' | 'after'>('before');
  zoom = $state(1);
  info = $state.raw<ProjectInfo | null>(null);
  folders = $state<string[]>([]);
  openError = $state<ErrorInfo | null>(null);
  ffmpeg = $state<ErrorInfo | null>(null);
  ffmpegPlan = $state<FFmpegPlan | null>(null);
  ffmpegLines = $state<string[]>([]);
  options = $state<Options>({ ...FALLBACK_OPTIONS });
  keepPaths = $state(false);
  optionError = $state<{ field: string; error: ErrorInfo } | null>(null);
  progress = $state<Progress>(newProgress(0));
  analyzed = new SvelteSet<string>();
  active = $state('');
  result = $state.raw<Result | null>(null);
  missing = $state<MissingAsk | null>(null);
  toasts = $state<Toast[]>([]);
  selected = $state<string | null>(null);
  hidden = $state<Partial<Record<FileResultKind, boolean>>>({});
  query = $state('');
  log = $state<string[]>([]);
  busy = $state('');
  about = $state<AboutInfo | null>(null);
  cliTool = $state<CLITool | null>(null);
  saved = $state('');
  now = $state(Date.now());
  saveState = $state<'' | 'saving' | 'saved' | 'error'>('');

  private viewChosen = false;
  private searchNote = '';
  private tick: ReturnType<typeof setInterval> | null = null;
  private saveTimer: ReturnType<typeof setTimeout> | null = null;
  private toastId = 0;

  async init(): Promise<void> {
    const b = await loadBackend();
    if (!b) {
      this.outside = true;
      return;
    }
    this.backend = b;
    b.on('sync:event', (d) => this.onEvent(d as SyncEvent));
    b.on('media:search', (d) => this.onSearch(d as SearchReport));
    b.on('media:missing', (d) => this.onMissing(d as MissingFile[]));
    b.on('ffmpeg:install', (d) => this.onFFmpegLine(String(d)));
    b.onFileDrop((paths) => this.onDrop(paths));
    try {
      const s = await b.startup();
      this.startup = s;
      this.options = { ...s.options };
      this.keepPaths = s.keepPaths;
      this.ffmpeg = s.ffmpeg ?? null;
      await this.changeLanguage(s.language || language());
      if (this.ffmpeg) void this.loadFFmpegPlan();
      if (s.ffmpegOld) this.toast('warn', t('ffmpeg.old', { version: s.ffmpegOld.version, min: s.ffmpegOld.min }));
      if (s.settingsError) this.toastError(s.settingsError, t('toast.settingsError'));
    } catch (e) {
      this.toastError(failed(e));
    }
    this.ready = true;
  }

  async changeLanguage(id: string): Promise<void> {
    const v = setLanguage(id);
    try {
      const e = await this.backend?.setLanguage(v, dialogTexts());
      if (e) this.logLine('gui.json', e);
    } catch (e) {
      this.logLine('gui.json', String(e));
    }
  }

  private async run<T>(op: string, f: (b: Backend) => Promise<T>): Promise<T | null> {
    if (!this.backend || this.busy) return null;
    this.busy = op;
    try {
      return await f(this.backend);
    } catch (e) {
      this.toastError(failed(e));
      return null;
    } finally {
      this.busy = '';
    }
  }

  get syncing(): boolean {
    return this.phase === 'syncing';
  }

  get canSync(): boolean {
    return !!this.info && !this.busy && !this.ffmpeg && this.ready;
  }

  async openDialog(): Promise<void> {
    const r = await this.run('open', (b) => b.chooseTimeline());
    if (r) this.applyOpen(r);
  }

  async openPath(path: string): Promise<void> {
    const r = await this.run('open', (b) => b.open(path));
    if (r) this.applyOpen(r);
  }

  onDrop(paths: string[]): void {
    if (this.busy) return;
    if (paths.length !== 1) {
      this.toast('warn', t('toast.dropOne'));
      return;
    }
    void this.openPath(paths[0]);
  }

  applyOpen(r: ProjectReply): void {
    this.folders = r.folders ?? [];
    if (r.error) {
      if (r.error.kind === 'busy') return this.toastError(r.error);
      this.openError = r.error;
      this.nav = 'import';
      return;
    }
    if (!r.info) return;
    this.openError = null;
    this.info = r.info;
    this.result = null;
    this.phase = 'ready';
    this.selected = null;
    this.hidden = {};
    this.saved = '';
    this.progress = newProgress(Date.now());
    this.analyzed.clear();
    this.active = '';
    if (r.options) this.options = { ...r.options };
    this.keepPaths = r.keepPaths;
    this.optionError = null;
    if (!this.viewChosen) this.view = 'files';
    this.ba = 'before';
    this.zoom = 1;
    if (r.settingsError) this.toastError(r.settingsError, t('toast.settingsError'));
  }

  async addFolder(): Promise<void> {
    const r = await this.run('search', (b) => b.addFolder());
    if (r) this.applySearch(r);
  }

  async removeFolder(dir: string): Promise<void> {
    if (!this.backend || this.busy) return;
    this.folders = await this.backend.removeFolder(dir);
  }

  async locate(): Promise<void> {
    const r = await this.run('search', (b) => b.locate());
    if (r) this.applySearch(r);
  }

  private applySearch(r: ProjectReply): void {
    this.missing = null;
    this.folders = r.folders ?? this.folders;
    if (r.info) this.info = r.info;
    if (r.error && r.error.kind !== 'cancelled') this.toastError(r.error);
  }

  onSearch(rep: SearchReport): void {
    this.logLine('search', rep);
    const roots = rep.roots ?? [];
    const folder = roots.length > 1 ? `${roots[0]} (+${roots.length - 1})` : (roots[0] ?? '');
    if (rep.badRoots?.length) this.searchNote = t('missing.badRoot', { folder: rep.badRoots.join(', ') });
    else if (rep.found === 0 && rep.answer) this.searchNote = t('missing.nothingNew', { folder });
    else this.searchNote = '';
    if (rep.found > 0) {
      this.toast('ok', folder ? t('toast.found', { found: rep.found, before: rep.before, folder }) : t('toast.foundAnswer', { found: rep.found, before: rep.before }));
    }
    if (rep.ambiguous?.length) {
      const list = rep.ambiguous.map((a) => `${a.file.name}\n  ${a.candidates.join('\n  ')}`).join('\n');
      this.toast('warn', t('toast.ambiguous', { n: rep.ambiguous.length }), undefined, list,
        [{ label: t('toast.copyDetails'), run: () => this.copy(list) }]);
    }
  }

  onMissing(files: MissingFile[]): void {
    this.missing = { files, note: this.searchNote };
    this.searchNote = '';
  }

  answer(action: 'skip' | 'cancel'): void {
    this.missing = null;
    void this.backend?.answer({ action });
  }

  async chooseCacheFolder(): Promise<void> {
    if (!this.backend || this.busy) return;
    const dir = await this.backend.chooseFolder();
    if (dir) this.setOption('cacheDir', dir);
  }

  async answerFolder(): Promise<void> {
    if (!this.backend) return;
    const dir = await this.backend.chooseFolder();
    if (!dir) return;
    this.missing = null;
    await this.backend.answer({ action: 'folder', folder: dir });
  }

  async synchronize(): Promise<void> {
    if (!this.canSync) return;
    this.phase = 'syncing';
    this.progress = newProgress(Date.now());
    this.analyzed.clear();
    this.active = '';
    this.selected = null;
    this.optionError = null;
    this.startTick();
    const r = await this.run('sync', (b) => b.sync($state.snapshot(this.options) as Options));
    this.stopTick();
    if (!r) {
      this.phase = this.result ? 'done' : 'ready';
      return;
    }
    if (r.info) this.info = r.info;
    if (r.folders) this.folders = r.folders;
    if (r.error) {
      if (r.error.kind === 'cancelled') {
        this.phase = 'cancelled';
        this.toast('warn', t('toast.cancelled'), t('toast.cancelledText'));
        return;
      }
      this.phase = this.result ? 'done' : 'ready';
      if (r.field) {
        this.optionError = { field: r.field, error: r.error };
        this.nav = 'settings';
      }
      this.toastError(r.error);
      return;
    }
    this.applyResult({ summary: r.summary!, files: r.files ?? [] });
  }

  applyResult(res: Result): void {
    this.result = res;
    this.phase = 'done';
    this.ba = 'after';
    this.hidden = {};
    this.zoom = 1;
    if (!this.viewChosen) this.view = 'timeline';
  }

  stop(): void {
    if (this.missing) this.answer('cancel');
    if (this.busy === 'sync' || this.busy === 'search' || this.busy === 'ffmpeg') void this.backend?.cancel();
  }

  onEvent(ev: SyncEvent): void {
    this.logLine('event', ev);
    if (this.phase !== 'syncing') return;
    this.progress = step(this.progress, ev);
    if (ev.stage === 'features' && ev.file) {
      this.analyzed.add(ev.file);
      this.active = ev.file;
    }
  }

  private startTick(): void {
    this.stopTick();
    this.now = Date.now();
    this.tick = setInterval(() => (this.now = Date.now()), 500);
  }

  private stopTick(): void {
    if (this.tick) clearInterval(this.tick);
    this.tick = null;
    this.now = Date.now();
  }

  async saveTimeline(format: string): Promise<void> {
    const r = await this.run('save', (b) => b.saveTimeline(format, this.keepPaths));
    if (!r) return;
    if (r.error) return this.toastError(r.error);
    const path = r.path ?? '';
    this.saved = path;
    this.toast('ok', t('toast.saved', { name: baseName(path) }), dirName(path), undefined,
      [{ label: t('toast.showInFolder'), run: () => void this.backend?.showInFolder(path) }]);
    if (format === 'drt') {
      this.toast('warn', t('toast.drtTitle'), t('toast.drtText'), undefined,
        [{ label: t('toast.saveReport'), run: () => void this.saveReport() }]);
    }
  }

  async saveReport(): Promise<void> {
    const r = await this.run('save', (b) => b.saveReport());
    if (!r) return;
    if (r.error) return this.toastError(r.error);
    const path = r.path ?? '';
    this.toast('ok', t('toast.reportSaved', { name: baseName(path) }), dirName(path), undefined,
      [{ label: t('toast.showInFolder'), run: () => void this.backend?.showInFolder(path) }]);
  }

  setOption<K extends keyof Options>(key: K, value: Options[K]): void {
    this.options[key] = value;
    this.queueSave();
  }

  setKeepPaths(v: boolean): void {
    this.keepPaths = v;
    this.queueSave();
  }

  resetOptions(): void {
    if (this.startup) this.options = { ...this.startup.defaults };
    this.keepPaths = false;
    this.queueSave();
  }

  private queueSave(): void {
    if (this.saveTimer) clearTimeout(this.saveTimer);
    this.saveState = 'saving';
    this.saveTimer = setTimeout(async () => {
      if (!this.backend) return;
      try {
        const c = await this.backend.saveSettings($state.snapshot(this.options) as Options, this.keepPaths);
        this.optionError = c.error ? { field: c.field ?? '', error: c.error } : null;
        this.saveState = c.error ? 'error' : 'saved';
      } catch (e) {
        this.saveState = 'error';
        this.toastError(failed(e));
      }
    }, 400);
  }

  setView(v: View): void {
    this.view = v;
    this.viewChosen = true;
  }

  setNav(n: Nav): void {
    this.nav = n;
    if (n === 'about' || n === 'dev') void this.loadAbout();
  }

  toggleHidden(r: FileResultKind): void {
    this.hidden = { ...this.hidden, [r]: !this.hidden[r] };
  }

  async loadAbout(): Promise<void> {
    try {
      this.about = (await this.backend?.about()) ?? null;
      this.cliTool = (await this.backend?.cliTool()) ?? null;
    } catch (e) {
      this.toastError(failed(e));
    }
  }

  async clearCache(): Promise<void> {
    const r = await this.run('clear-cache', (b) => b.clearCache());
    if (!r) return;
    if (this.about) this.about = { ...this.about, cache: r.cache };
    if (r.error) return this.toastError(r.error);
    this.toast('ok', t('about.cleared', { n: r.cleared.removed, size: bytes(r.cleared.bytes) }));
  }

  async installCliTool(): Promise<void> {
    await this.changeCliTool('install');
  }

  async removeCliTool(): Promise<void> {
    await this.changeCliTool('remove');
  }

  private async changeCliTool(action: 'install' | 'remove'): Promise<void> {
    const r = await this.run('cli-tool', (b) =>
      action === 'install' ? b.installCliTool(t('about.cliPromptInstall')) : b.removeCliTool(t('about.cliPromptRemove')));
    if (!r) return;
    this.cliTool = r.tool;
    if (r.error?.kind === 'cancelled') return this.toast('warn', t('about.cliCancelled'));
    if (r.error) return this.toastError(r.error);
    this.toast('ok', action === 'install' ? t('about.cliInstalledToast') : t('about.cliRemovedToast'),
      action === 'install' ? r.tool.link : undefined);
  }

  async loadFFmpegPlan(): Promise<void> {
    try {
      this.ffmpegPlan = (await this.backend?.ffmpegPlan()) ?? null;
    } catch (e) {
      this.logLine('ffmpeg plan', String(e));
    }
  }

  onFFmpegLine(line: string): void {
    this.ffmpegLines = [...this.ffmpegLines.slice(-199), line];
    this.logLine('ffmpeg', line);
  }

  async installFFmpeg(): Promise<void> {
    this.ffmpegLines = [];
    const r = await this.run('ffmpeg', (b) => b.installFFmpeg());
    if (!r) return;
    this.ffmpeg = r.ffmpeg ?? null;
    if (!this.ffmpeg) return this.toast('ok', t('ffmpeg.found'));
    if (r.error?.kind === 'cancelled') return this.toast('warn', t('ffmpeg.stopped'));
    this.toastError(r.error ?? this.ffmpeg, t('ffmpeg.failed'));
  }

  async checkFFmpeg(): Promise<void> {
    if (!this.backend) return;
    try {
      this.ffmpeg = await this.backend.checkFFmpeg();
      if (this.ffmpeg) this.toast('warn', t('ffmpeg.still'));
      else this.toast('ok', t('ffmpeg.found'));
    } catch (e) {
      this.toastError(failed(e));
    }
  }

  async openLogs(): Promise<void> {
    await this.backend?.openLogs();
  }

  logLine(kind: string, v: unknown): void {
    const line = kind === 'event' ? JSON.stringify(v) : `${kind} ${JSON.stringify(v)}`;
    const next = this.log.length >= LOG_MAX ? this.log.slice(-LOG_MAX + 1) : this.log.slice();
    next.push(line);
    this.log = next;
  }

  toast(level: Toast['level'], title: string, text?: string, details?: string, actions?: ToastAction[]): void {
    const id = ++this.toastId;
    this.toasts = [...this.toasts.slice(-3), { id, level, title, text, details, actions }];
    if (level !== 'err') setTimeout(() => this.dismiss(id), 8000);
  }

  dismiss(id: number): void {
    this.toasts = this.toasts.filter((x) => x.id !== id);
  }

  toastError(e: ErrorInfo, title?: string): void {
    const k = has(`error.${e.kind}.title`) ? e.kind : 'failed';
    const details = [e.message, e.hint].filter(Boolean).join('\n');
    this.toast('err', title ?? t(`error.${k}.title`), t(`error.${k}.hint`), details,
      [{ label: t('toast.copyDetails'), run: () => this.copy(details) }]);
  }

  async copy(text: string): Promise<void> {
    await this.backend?.copyText(text);
    this.toast('info', t('toast.copied'));
  }
}

export const app = new AppState();

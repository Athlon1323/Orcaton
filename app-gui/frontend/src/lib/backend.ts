import type {
  AboutInfo, Answer, CacheReply, CLITool, CLIToolReply, DialogTexts, ErrorInfo, FFmpegPlan, InstallReply, OptionCheck, Options,
  PathReply, ProjectReply, StartupInfo, SyncReply,
} from './types';

export interface Backend {
  readonly kind: 'wails' | 'mock';
  startup(): Promise<StartupInfo>;
  checkFFmpeg(): Promise<ErrorInfo | null>;
  ffmpegPlan(): Promise<FFmpegPlan>;
  installFFmpeg(): Promise<InstallReply>;
  setLanguage(lang: string, d: DialogTexts): Promise<ErrorInfo | null>;
  chooseTimeline(): Promise<ProjectReply | null>;
  open(path: string): Promise<ProjectReply>;
  addFolder(): Promise<ProjectReply | null>;
  removeFolder(dir: string): Promise<string[]>;
  locate(): Promise<ProjectReply>;
  chooseFolder(): Promise<string>;
  answer(a: Answer): Promise<void>;
  sync(o: Options): Promise<SyncReply>;
  cancel(): Promise<void>;
  saveTimeline(format: string, keepPaths: boolean): Promise<PathReply | null>;
  saveReport(): Promise<PathReply | null>;
  showInFolder(path: string): Promise<void>;
  openLogs(): Promise<void>;
  copyText(text: string): Promise<void>;
  validateOptions(o: Options): Promise<OptionCheck>;
  saveSettings(o: Options, keepPaths: boolean): Promise<OptionCheck>;
  about(): Promise<AboutInfo>;
  clearCache(): Promise<CacheReply>;
  cliTool(): Promise<CLITool>;
  installCliTool(prompt: string): Promise<CLIToolReply>;
  removeCliTool(prompt: string): Promise<CLIToolReply>;
  on(event: string, cb: (data: unknown) => void): () => void;
  onFileDrop(cb: (paths: string[]) => void): void;
}

type GoFn = (...args: unknown[]) => Promise<unknown>;
interface WailsWindow {
  go?: { main?: { App?: Record<string, GoFn> } };
  runtime?: {
    EventsOnMultiple(name: string, cb: (...data: unknown[]) => void, max: number): () => void;
    OnFileDrop(cb: (x: number, y: number, paths: string[]) => void, useDropTarget: boolean): void;
  };
}

const w = () => window as unknown as WailsWindow;

function call<T>(name: string, ...args: unknown[]): Promise<T> {
  const fn = w().go?.main?.App?.[name];
  if (!fn) return Promise.reject(new Error(`Go method App.${name} is not bound`));
  return fn(...args) as Promise<T>;
}

const orNull = <T>(v: T | null | undefined): T | null => (v === undefined ? null : v);

const wails: Backend = {
  kind: 'wails',
  startup: () => call('Startup'),
  checkFFmpeg: async () => orNull(await call<ErrorInfo | null>('CheckFFmpeg')),
  ffmpegPlan: () => call('FFmpegPlan'),
  installFFmpeg: () => call('InstallFFmpeg'),
  setLanguage: async (lang, d) => orNull(await call<ErrorInfo | null>('SetLanguage', lang, d)),
  chooseTimeline: async () => orNull(await call<ProjectReply | null>('ChooseTimeline')),
  open: (path) => call('Open', path),
  addFolder: async () => orNull(await call<ProjectReply | null>('AddFolder')),
  removeFolder: (dir) => call('RemoveFolder', dir),
  locate: () => call('Locate'),
  chooseFolder: () => call('ChooseFolder'),
  answer: (a) => call('Answer', a),
  sync: (o) => call('Sync', o),
  cancel: () => call('Cancel'),
  saveTimeline: async (format, keepPaths) => orNull(await call<PathReply | null>('SaveTimeline', format, keepPaths)),
  saveReport: async () => orNull(await call<PathReply | null>('SaveReport')),
  showInFolder: (path) => call('ShowInFolder', path),
  openLogs: () => call('OpenLogs'),
  copyText: (text) => call('CopyText', text),
  validateOptions: (o) => call('ValidateOptions', o),
  saveSettings: (o, keepPaths) => call('SaveSettings', o, keepPaths),
  about: () => call('About'),
  clearCache: () => call('ClearCache'),
  cliTool: () => call('CLITool'),
  installCliTool: (prompt) => call('InstallCLITool', prompt),
  removeCliTool: (prompt) => call('RemoveCLITool', prompt),
  on: (event, cb) => w().runtime!.EventsOnMultiple(event, (...d) => cb(d[0]), -1),
  onFileDrop: (cb) => w().runtime!.OnFileDrop((_x, _y, paths) => cb(paths ?? []), false),
};

const hasWails = () => !!w().go?.main?.App && !!w().runtime;

export async function loadBackend(): Promise<Backend | null> {
  if (import.meta.env.MODE === 'mock') {
    const load = import.meta.glob<{ createMock: () => Backend }>('./mock.ts')['./mock.ts'];
    return load ? (await load()).createMock() : null;
  }
  for (let i = 0; i < 40 && !hasWails(); i++) await new Promise((r) => setTimeout(r, 50));
  return hasWails() ? wails : null;
}

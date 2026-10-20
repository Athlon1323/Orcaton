
export type ErrorKind =
  | 'cancelled' | 'no_ffmpeg' | 'not_found' | 'bad_format' | 'bad_options'
  | 'bad_output' | 'write' | 'failed' | 'busy' | 'no_project';

export interface ErrorInfo {
  kind: ErrorKind;
  message: string;
  hint?: string;
}

export interface Options {
  unsynced: string;
  orderBy: string;
  singles: string;
  unsyncedLabel: string;
  timeLabel: string;
  minConfidence: number;
  minRatio: number;
  autoThreshold: boolean;
  minOverlapSec: number;
  maxOffsetSec: number;
  audioMode: string;
  fileOrder: boolean;
  intraLane: boolean;
  recordTime: boolean;
  maxGapSec: number;
  keepGapSec: number;
  jobs: number;
  threads: number;
  memLimitMB: number;
  cacheDir: string;
}

export interface StartupInfo {
  version: string;
  apiVersion: string;
  os: string;
  ffmpeg?: ErrorInfo;
  ffmpegOld?: FFmpegInfo;
  options: Options;
  defaults: Options;
  keepPaths: boolean;
  settingsPath: string;
  settingsError?: ErrorInfo;
  unsyncedModes: string[];
  orderModes: string[];
  singlesModes: string[];
  audioModes: string[];
  logDir: string;
  language: string;
  languages: string[];
}

export interface DialogTexts {
  openTimeline: string;
  timelineFilter: string;
  allFiles: string;
  chooseFolder: string;
  saveTimeline: string;
  saveReport: string;
}

export type FileState = 'ok' | 'missing' | 'offline' | 'nomedia';

export interface MediaFile {
  name: string;
  path: string;
  originalPath?: string;
  lane: string;
  camera: string;
  durationSec: number;
  hasAudio: boolean;
  state: FileState;
  tracks: string[];
  startSec: number;
  startTC: string;
}

export interface Camera {
  id: string;
  lane: string;
  folder: string;
  files: number;
  tracks: string[];
  title: string;
}

export interface SequenceInfo {
  name: string;
  fps: string;
  fpsNum: number;
  fpsDen: number;
  startSec: number;
  startTC: string;
  durationSec: number;
  tracks: number;
  clips: number;
}

export interface OutputFormat {
  name: string;
  title: string;
  extensions: string[];
}

export interface ProjectInfo {
  path: string;
  name: string;
  format: string;
  sequences: SequenceInfo[];
  cameras: Camera[];
  files: MediaFile[];
  missing: number;
  offline: number;
  outputFormats: OutputFormat[];
}

export interface ProjectReply {
  info?: ProjectInfo;
  folders: string[];
  options?: Options;
  keepPaths: boolean;
  settingsError?: ErrorInfo;
  error?: ErrorInfo;
}

export type Stage = 'search' | 'record_time' | 'features' | 'pairs' | 'refine' | 'solve' | 'layout' | 'write';

export interface SyncEvent {
  stage: Stage;
  title: string;
  done: number;
  total: number;
  file?: string;
}

export interface MissingFile {
  name: string;
  path: string;
  originalPath: string;
}

export interface Ambiguous {
  file: MissingFile;
  candidates: string[];
}

export interface SearchReport {
  answer: boolean;
  roots: string[] | null;
  badRoots: string[] | null;
  before: number;
  found: number;
  ambiguous: Ambiguous[] | null;
  notFound: MissingFile[] | null;
}

export interface Answer {
  action: 'folder' | 'skip' | 'cancel';
  folder?: string;
}

export type FileResultKind = 'audio' | 'clock' | 'unmatched' | 'offline';

export interface FileResult {
  name: string;
  path: string;
  lane: string;
  camera: string;
  result: FileResultKind;
  status: string;
  positionSec?: number;
  durationSec: number;
  sourceInSec: number;
  group: number;
  partner?: string;
  confidence?: number;
  ratio?: number;
  recordStart?: string;
  reason?: string;
  reasonCode?: string;
  tracks: string[];
  startSec: number;
  positionTC?: string;
}

export interface Resources {
  ramMB: number;
  memLimitMB: number;
  threads: number;
  jobs: number;
  disk: string;
  peakMB: number;
}

export interface Summary {
  files: number;
  audio: number;
  clock: number;
  unmatched: number;
  offline: number;
  seconds: number;
  threshold: number;
  reasons: Record<string, number> | null;
  resources: Resources;
}

export interface SyncReply {
  summary?: Summary;
  files?: FileResult[];
  info?: ProjectInfo;
  folders?: string[];
  error?: ErrorInfo;
  field?: string;
}

export interface PathReply {
  path?: string;
  error?: ErrorInfo;
}

export interface OptionCheck {
  field?: string;
  error?: ErrorInfo;
}

export interface FFmpegInfo {
  ffmpeg: string;
  ffprobe: string;
  version: string;
  old: boolean;
  min: string;
}

export interface FFmpegPlan {
  os: string;
  tool: string;
  command: string;
  button: boolean;
  admin: boolean;
  problem: '' | 'no_winget' | 'no_brew' | 'no_pkexec' | 'no_manager';
}

export interface InstallReply {
  ffmpeg?: ErrorInfo;
  error?: ErrorInfo;
}

export interface CacheInfo {
  path: string;
  exists: boolean;
  entries: number;
  bytes: number;
  partial: number;
}

export interface AboutInfo {
  version: string;
  apiVersion: string;
  ffmpeg: FFmpegInfo;
  ffmpegError?: ErrorInfo;
  cache: CacheInfo;
  settingsPath: string;
  logDir: string;
}

export interface CLITool {
  available: boolean;
  installed: boolean;
  link: string;
  target: string;
  other?: string;
  problem?: '' | 'not_in_applications' | 'no_binary' | 'occupied';
}

export interface CLIToolReply {
  tool: CLITool;
  error?: ErrorInfo;
}

export interface CacheReply {
  cleared: { removed: number; bytes: number; errors: string[] | null };
  cache: CacheInfo;
  error?: ErrorInfo;
}

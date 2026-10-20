import en from '../i18n/en.json';
import de from '../i18n/de.json';
import ru from '../i18n/ru.json';
import ka from '../i18n/ka.json';
import { lang } from './lang.svelte';
import type { DialogTexts } from './types';

type Dict = Record<string, string>;
const dicts: Record<string, Dict> = { en, de, ru, ka };

export const LANGUAGES = [
  { id: 'en', title: 'English' },
  { id: 'de', title: 'Deutsch' },
  { id: 'ru', title: 'Русский' },
  { id: 'ka', title: 'ქართული' },
];

export const DEFAULT_LANGUAGE = 'en';

export function setLanguage(id: string): string {
  const v = id in dicts ? id : DEFAULT_LANGUAGE;
  lang.id = v;
  if (typeof document !== 'undefined') document.documentElement.lang = v;
  return v;
}

export function language(): string {
  return lang.id;
}

export function t(key: string, params?: Record<string, string | number>): string {
  const s = dicts[lang.id]?.[key] ?? en[key as keyof typeof en] ?? key;
  if (!params) return s;
  return s.replace(/\{(\w+)\}/g, (m, k: string) => (k in params ? String(params[k]) : m));
}

export function has(key: string): boolean {
  return key in en;
}

export function keys(id = DEFAULT_LANGUAGE): string[] {
  return Object.keys(dicts[id] ?? {});
}

export function dict(id: string): Dict | undefined {
  return dicts[id];
}

export function dialogTexts(): DialogTexts {
  return {
    openTimeline: t('dialog.openTimeline'),
    timelineFilter: t('dialog.timelineFilter'),
    allFiles: t('dialog.allFiles'),
    chooseFolder: t('dialog.chooseFolder'),
    saveTimeline: t('dialog.saveTimeline'),
    saveReport: t('dialog.saveReport'),
  };
}

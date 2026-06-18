import { format, parseISO } from 'date-fns';

export const formatHMS = (seconds) => {
  const s = Math.max(0, Math.floor(seconds));
  const h = Math.floor(s / 3600).toString().padStart(2, '0');
  const m = Math.floor((s % 3600) / 60).toString().padStart(2, '0');
  const ss = (s % 60).toString().padStart(2, '0');
  return `${h}:${m}:${ss}`;
};

export const formatDuration = (seconds) => {
  const s = Math.max(0, Math.floor(seconds));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const ss = s % 60;
  return `${h}h ${m}m ${ss}s`;
};

export const formatDate = (isoDate) => {
  if (!isoDate) return '';
  return format(parseISO(isoDate), 'EEE, MMM d');
};

export const formatClock = (ts) => {
  if (!ts) return '';
  return format(new Date(ts), 'h:mm:ss a');
};

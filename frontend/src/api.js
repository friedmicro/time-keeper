const BASE = 'http://' + window.location.hostname + ":" + window.location.port;

const jsonFetch = async (path, options = {}) => {
  const res = await fetch(BASE + path, options);
  return res.json();
};

export const getCurrent = () => jsonFetch('/api/current');

export const startTimer = (category = 'uncategorized') => jsonFetch('/api/start', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ category: category || 'uncategorized' }),
});

export const stopTimer = () => jsonFetch('/api/stop', { method: 'POST' });

export const getWeek = () => jsonFetch('/api/week');

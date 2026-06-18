import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getCurrent, startTimer, stopTimer } from '../api.js';
import { formatClock, formatHMS } from '../format.js';

const Timer = () => {
  const [running, setRunning] = useState(false);
  const [startedAt, setStartedAt] = useState(null);
  const [, setTick] = useState(0); // bumped each second to re-render the clock
  const [category, setCategory] = useState('');
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [flash, setFlash] = useState(null);

  // Poll current state on mount and keep a 1s clock running.
  useEffect(() => {
    let cancelled = false;
    getCurrent()
      .then((data) => {
        if (cancelled || !data || !data.running) return;
        setStartedAt(new Date(data.started_at).getTime());
        setRunning(true);
      })
      .catch((e) => console.error('Failed to check status:', e));

    const id = setInterval(() => setTick((t) => t + 1), 1000);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, []);

  const elapsed = running && startedAt ? Math.floor((Date.now() - startedAt) / 1000) : 0;

  const showFlash = (type, msg) => {
    setFlash({ type, msg });
    window.setTimeout(() => setFlash(null), 3500);
  };

  const handleStart = async () => {
    try {
      const data = await startTimer(category);
      if (data.status === 'ok') {
        setStartedAt(Date.now());
        setRunning(true);
        showFlash('ok', 'Timer started!');
      } else {
        showFlash('err', data.message || 'Failed to start timer');
      }
    } catch (e) {
      showFlash('err', 'Failed to start timer');
    }
  }

  const handleStop = async () => {
    try {
      const data = await stopTimer();
      if (data.status === 'ok') {
        setRunning(false);
        setStartedAt(null);
        showFlash('ok', 'Timer stopped and logged!');
      } else {
        showFlash('err', data.message || 'Failed to stop timer');
      }
    } catch (e) {
      showFlash('err', 'Failed to stop timer');
    }
  }

  return (
    <div className="flex min-h-screen flex-col items-center bg-bg px-8 py-8">
      <h1 className="mb-8 text-3xl font-bold text-accent">Time Tracker</h1>

      {flash && (
        <div
          className={`mb-4 rounded-lg px-6 py-3 font-medium ${
            flash.type === 'ok' ? 'bg-mint text-bg' : 'bg-accent text-white'
          }`}
        >
          {flash.msg}
        </div>
      )}

      <div className="mb-8 rounded-2xl bg-card px-12 py-8 text-center shadow-xl">
        <div className={`mb-4 text-sm ${running ? 'text-mint' : 'text-muted'}`}>
          {running ? 'Running' : 'Stopped'}
        </div>
        <div className="my-4 text-7xl font-light tabular-nums text-white">
          {formatHMS(elapsed)}
        </div>
        {running && startedAt && (
          <div className="mb-2 text-xs text-muted">
            Started at {formatClock(new Date(startedAt))}
          </div>
        )}

        <div>
          <button
            onClick={handleStart}
            disabled={running}
            className="m-1 rounded-lg bg-mint px-8 py-3 text-lg text-bg transition enabled:hover:scale-105 enabled:active:scale-95 disabled:cursor-not-allowed disabled:opacity-40"
          >
            Start
          </button>
          <button
            onClick={handleStop}
            disabled={!running}
            className="m-1 rounded-lg bg-accent px-8 py-3 text-lg text-white transition enabled:hover:scale-105 enabled:active:scale-95 disabled:cursor-not-allowed disabled:opacity-40"
          >
            Stop
          </button>
        </div>

        {showAdvanced && (
          <input
            type="text"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            placeholder="What are you working on?"
            className="mx-auto mt-6 block rounded-md border-4 border-panel bg-card px-1 py-0.5 text-white placeholder:text-muted focus:outline-none"
          />
        )}
      </div>

      <nav className="mt-4 text-sm">
        <Link to="/" className="mx-4 text-mint hover:underline">
          Timer
        </Link>
        <Link to="/summary" className="mx-4 text-mint hover:underline">
          Daily Summary
        </Link>
        <a
          href="#"
          onClick={(e) => {
            e.preventDefault();
            setShowAdvanced((v) => !v);
          }}
          className="mx-4 text-mint hover:underline"
        >
          Advanced
        </a>
      </nav>
    </div>
  );
};

export default Timer;

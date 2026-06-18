import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { getWeek } from '../api.js';
import { formatDate } from '../format.js';

const Summary = () => {
  const [weekTotal, setWeekTotal] = useState('0h 0m 0s');
  const [days, setDays] = useState([]);
  const [categories, setCategories] = useState([]);
  const [error, setError] = useState(null);
  const [showCategories, setShowCategories] = useState(
    () => localStorage.getItem('showCategories') === 'true',
  );

  useEffect(() => {
    getWeek()
      .then((data) => {
        if (data.error) {
          setError(data.error);
          return;
        }
        setWeekTotal(data.week_total || '0h 0m 0s');
        setDays(data.days || []);
        setCategories(data.categories || []);
      })
      .catch(() => setError('Failed to load summary'));
  }, []);

  useEffect(() => {
    localStorage.setItem('showCategories', String(showCategories));
  }, [showCategories]);

  return (
    <div className="min-h-screen bg-bg px-8 py-8 text-eee">
      <h1 className="mb-8 text-3xl font-bold text-accent">Daily Summary</h1>

      <div className="mb-6 max-w-[600px] rounded-xl bg-panel px-6 py-4 text-lg text-mint">
        <strong>Week total:</strong> {weekTotal}
      </div>

      {error && (
        <p className="max-w-[600px] rounded-xl bg-card px-6 py-4 text-white">{error}</p>
      )}

      {!showCategories ? (
        days.length > 0 ? (
          <table className="w-full max-w-[600px] overflow-hidden rounded-xl border-collapse bg-card text-left shadow-xl">
            <thead>
              <tr className="bg-panel text-accent">
                <th className="px-6 py-4 font-semibold">Date</th>
                <th className="px-6 py-4 font-semibold">Total Time</th>
              </tr>
            </thead>
            <tbody>
              {days.map((d) => (
                <tr key={d.date} className="border-b border-panel last:border-b-0">
                  <td className="px-6 py-4 text-white">{formatDate(d.date)}</td>
                  <td className="px-6 py-4 text-white">{d.total}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="max-w-[600px] rounded-xl bg-card py-8 text-center text-muted">
            No time entries yet. Start tracking!
          </p>
        )
      ) : categories.length > 0 ? (
        <table className="w-full max-w-[600px] overflow-hidden rounded-xl border-collapse bg-card text-left shadow-xl">
          <thead>
            <tr className="bg-panel text-accent">
              <th className="px-6 py-4 font-semibold">Date</th>
              <th className="px-6 py-4 font-semibold">Category</th>
              <th className="px-6 py-4 font-semibold">Total Time</th>
            </tr>
          </thead>
          <tbody>
            {categories.map((c, i) => (
              <tr key={`${c.date}-${c.category}-${i}`} className="border-b border-panel last:border-b-0">
                <td className="px-6 py-4 text-white">{formatDate(c.date)}</td>
                <td className="px-6 py-4 text-white">{c.category}</td>
                <td className="px-6 py-4 text-white">{c.total}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}

      <label className="mb-6 mt-6 block max-w-[600px] cursor-pointer rounded-xl bg-panel px-6 py-4 text-lg text-mint">
        <input
          type="checkbox"
          checked={showCategories}
          onChange={(e) => setShowCategories(e.target.checked)}
          className="mr-2"
        />
        Show categories
      </label>

      <nav className="mt-4 text-sm">
        <Link to="/" className="mx-4 text-mint hover:underline">
          Timer
        </Link>
        <Link to="/summary" className="mx-4 text-mint hover:underline">
          Daily Summary
        </Link>
      </nav>
    </div>
  );
};

export default Summary;

import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Timer from './pages/Timer.jsx';
import Summary from './pages/Summary.jsx';

const App = () => {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Timer />} />
        <Route path="/summary" element={<Summary />} />
        <Route path="*" element={<Timer />} />
      </Routes>
    </BrowserRouter>
  );
};

export default App;
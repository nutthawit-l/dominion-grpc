import { Routes, Route, Navigate } from 'react-router-dom'
import Lobby from './screens/Lobby'
import KingdomPicker from './screens/KingdomPicker'
import GameTable from './screens/GameTable'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Lobby />} />
      <Route path="/new" element={<KingdomPicker />} />
      <Route path="/game/:gameId" element={<GameTable />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

import {
  Plus,
  Trash2,
  Pencil,
  Check,
  X,
  Save,
  Play,
  Pause,
  Clock,
  Link2,
  Tag,
  FileText,
  Settings,
  Folder,
  ListTodo,
  BarChart3,
  RefreshCw,
  Download,
  Upload,
  Edit3,
  Eye,
  EyeOff,
  ChevronDown,
  ChevronUp,
  ChevronRight,
  Search,
  Filter,
  Calendar,
  History,
  MessageSquare,
  ListChecks,
  Square,
  SquareCheck,
  SquareX,
} from 'lucide-react'
import type { LucideProps } from 'lucide-react'

const map = {
  plus: Plus,
  add: Plus,
  trash: Trash2,
  delete: Trash2,
  remove: X,
  pencil: Pencil,
  edit: Edit3,
  check: Check,
  close: X,
  save: Save,
  play: Play,
  pause: Pause,
  clock: Clock,
  link: Link2,
  tag: Tag,
  file: FileText,
  settings: Settings,
  folder: Folder,
  tasks: ListTodo,
  reports: BarChart3,
  refresh: RefreshCw,
  download: Download,
  upload: Upload,
  eye: Eye,
  eyeOff: EyeOff,
  chevronDown: ChevronDown,
  chevronUp: ChevronUp,
  chevronRight: ChevronRight,
  search: Search,
  filter: Filter,
  calendar: Calendar,
  history: History,
  journal: MessageSquare,
  checklist: ListChecks,
  square: Square,
  checkSquare: SquareCheck,
  xSquare: SquareX,
} as const

export type IconName = keyof typeof map

export function Icon({ name, size = 16, ...props }: { name: IconName; size?: number } & LucideProps) {
  const Cmp = map[name]
  if (!Cmp) return null
  return <Cmp size={size} {...props} />
}

// Реэкспорт отдельных иконок, если нужен прямой импорт
export { Plus, Trash2, Pencil, Check, X, Save }

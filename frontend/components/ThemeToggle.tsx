type Props = {
  theme: 'light' | 'dark';
  onToggle: () => void;
};

export default function ThemeToggle({ theme, onToggle }: Props) {
  return (
    <button
      type="button"
      className="theme-toggle"
      onClick={onToggle}
      aria-label="Toggle theme"
      title={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
    >
      <span>{theme === 'dark' ? 'Dark' : 'Light'}</span>
      <strong>{theme === 'dark' ? '☾' : '☀'}</strong>
    </button>
  );
}

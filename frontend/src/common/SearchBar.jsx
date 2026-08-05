export default function SearchBar({
  value,
  onChange,
  onFocus,
  onClear,
  placeholder = 'Search sites, hostname, or IP…',
  id = 'global-search',
  autoFocus = false,
  variant = 'page',
}) {
  return (
    <div className={variant === 'nav' ? 'searchbar-nav' : 'searchbar-row'}>
      <label className={`searchbar searchbar-${variant}`} htmlFor={id}>
        <span className="searchbar-icon" aria-hidden="true">⌕</span>
        <input
          id={id}
          type="search"
          value={value}
          onChange={event => onChange(event.target.value)}
          onFocus={onFocus}
          placeholder={placeholder}
          autoComplete="off"
          autoFocus={autoFocus}
        />
        {value ? (
          <button
            type="button"
            className="searchbar-clear"
            onClick={onClear}
            aria-label="Clear search"
          >
            Clear
          </button>
        ) : null}
      </label>
    </div>
  )
}

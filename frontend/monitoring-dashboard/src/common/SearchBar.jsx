export default function SearchBar({ value, onChange }) {
  return (
    <div className="searchbar-row">
      <label className="searchbar" htmlFor="site-search">
        <span className="searchbar-icon" aria-hidden="true">⌕</span>
        <input
          id="site-search"
          type="search"
          value={value}
          onChange={event => onChange(event.target.value)}
          placeholder="Search by site name, type, status, or ID"
          autoComplete="off"
        />
      </label>
    </div>
  )
}

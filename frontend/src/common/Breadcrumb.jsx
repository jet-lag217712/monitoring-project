export default function Breadcrumb({ items }) {
  return (
    <nav className="breadcrumb" aria-label="Breadcrumb">
      {items.map((item, index) => {
        const isLast = index === items.length - 1
        return (
          <span key={item.label} className="breadcrumb-segment">
            {index > 0 && <span className="breadcrumb-sep" aria-hidden="true">›</span>}
            {isLast || !item.onClick ? (
              <span className="breadcrumb-current" aria-current={isLast ? 'page' : undefined}>
                {item.label}
              </span>
            ) : (
              <button type="button" className="breadcrumb-link" onClick={item.onClick}>
                {item.label}
              </button>
            )}
          </span>
        )
      })}
    </nav>
  )
}

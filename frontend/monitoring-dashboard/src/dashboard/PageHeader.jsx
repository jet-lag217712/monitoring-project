export default function PageHeader({ eyebrow, title, subtitle, rightContent }) {
  return (
    <div className="page-header">
      <div className="page-header-left">
        <div className="page-eyebrow">
          <span className="eyebrow-dot" />
          {eyebrow}
        </div>
        <h1 className="page-title">{title}</h1>
        <p className="page-sub">{subtitle}</p>
      </div>
      {rightContent}
    </div>
  )
}

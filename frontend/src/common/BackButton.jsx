export default function BackButton({ onClick, children = '← Back' }) {
  return (
    <button className="back-btn" onClick={onClick}>
      {children}
    </button>
  )
}

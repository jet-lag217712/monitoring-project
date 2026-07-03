export default function BackButton({ onClick, children = '← All Sites' }) {
  return (
    <button className="back-btn" onClick={onClick}>
      {children}
    </button>
  )
}

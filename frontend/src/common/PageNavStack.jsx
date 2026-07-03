import BackButton from './BackButton.jsx'
import Breadcrumb from './Breadcrumb.jsx'

export default function PageNavStack({ breadcrumbItems, onBack }) {
  return (
    <div className="page-nav-stack">
      <Breadcrumb items={breadcrumbItems} />
      {onBack && <BackButton onClick={onBack} />}
    </div>
  )
}

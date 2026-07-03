import SiteDetail from '../sites/SiteDetail.jsx'
import SitesGrid from '../sites/SitesGrid.jsx'

export default function DashboardPage({ dashboard }) {
  const {
    handleBack,
    handleSiteClick,
    lastUpdated,
    searchQuery,
    selectedSite,
    setSearchQuery,
    siteDetail,
    visibleSites,
    dataMode,
  } = dashboard

  if (selectedSite) {
    return (
      <SiteDetail
        data={siteDetail}
        onBack={handleBack}
      />
    )
  }

  return (
    <SitesGrid
      sites={visibleSites}
      onSiteClick={handleSiteClick}
      lastUpdated={lastUpdated}
      searchQuery={searchQuery}
      onSearchQueryChange={setSearchQuery}
      dataMode={dataMode}
    />
  )
}

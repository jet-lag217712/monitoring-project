import DeviceDetail from '../devices/DeviceDetail.jsx'
import SiteDetail from '../sites/SiteDetail.jsx'
import SitesGrid from '../sites/SitesGrid.jsx'

export default function DashboardPage({ dashboard }) {
  const {
    getSelectedInterfaceKey,
    handleBack,
    handleDeviceBack,
    handleDeviceClick,
    handleInterfaceSelect,
    handleSiteClick,
    lastUpdated,
    searchQuery,
    selectedDevice,
    selectedSite,
    setSearchQuery,
    siteDetail,
    visibleSites,
    dataMode,
  } = dashboard

  if (selectedDevice && selectedSite) {
    const device = siteDetail?.latest?.devices?.[selectedDevice]

    return (
      <DeviceDetail
        site={siteDetail}
        device={device}
        deviceIp={selectedDevice}
        selectedInterfaceKey={getSelectedInterfaceKey(selectedDevice)}
        onInterfaceSelect={handleInterfaceSelect}
        onNavigateAllSites={handleBack}
        onNavigateSite={handleDeviceBack}
      />
    )
  }

  if (selectedSite) {
    return (
      <SiteDetail
        data={siteDetail}
        onBack={handleBack}
        onDeviceClick={handleDeviceClick}
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

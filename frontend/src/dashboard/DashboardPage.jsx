import DeviceDetail from '../devices/DeviceDetail.jsx'
import SiteDetail from '../sites/SiteDetail.jsx'
import SitesGrid from '../sites/SitesGrid.jsx'

export default function DashboardPage({ dashboard }) {
  const {
    deviceDetail,
    deviceError,
    deviceLoading,
    getSelectedInterfaceKey,
    handleBack,
    handleDeviceBack,
    handleDeviceClick,
    handleInterfaceSelect,
    handleRenameLocation,
    handleSiteClick,
    lastUpdated,
    loadError,
    searchQuery,
    selectedDevice,
    selectedSite,
    setSearchQuery,
    siteDetail,
    visibleSites,
    dataMode,
  } = dashboard

  if (selectedDevice && selectedSite) {
    const fallback = siteDetail?.latest?.devices?.[selectedDevice]
    const device = deviceDetail ?? fallback

    return (
      <DeviceDetail
        site={siteDetail}
        device={device}
        deviceIp={selectedDevice}
        loading={deviceLoading && !device}
        error={deviceError}
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
        siteId={selectedSite}
        onBack={handleBack}
        onDeviceClick={handleDeviceClick}
        onRenameLocation={handleRenameLocation}
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
      loadError={loadError}
    />
  )
}

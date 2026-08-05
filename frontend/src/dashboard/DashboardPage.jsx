import DeviceDetail from '../devices/DeviceDetail.jsx'
import SearchTakeover from './SearchTakeover.jsx'
import SiteDetail from '../sites/SiteDetail.jsx'
import SitesGrid from '../sites/SitesGrid.jsx'

export default function DashboardPage({ dashboard }) {
  const {
    closeSearch,
    deviceDetail,
    deviceError,
    deviceLoading,
    getSelectedInterfaceKey,
    handleBack,
    handleDeviceBack,
    handleDeviceClick,
    handleInterfaceSelect,
    handleRenameLocation,
    handleSearchDeviceSelect,
    handleSiteClick,
    lastUpdated,
    loadError,
    searchDeviceHits,
    searchLoading,
    searchOpen,
    searchQuery,
    searchSiteHits,
    selectedDevice,
    selectedSite,
    setSearchQuery,
    siteDetail,
    sites,
    dataMode,
  } = dashboard

  if (searchOpen) {
    return (
      <SearchTakeover
        query={searchQuery}
        onQueryChange={setSearchQuery}
        onClose={closeSearch}
        siteHits={searchSiteHits}
        deviceHits={searchDeviceHits}
        loading={searchLoading}
        onSelectSite={handleSiteClick}
        onSelectDevice={handleSearchDeviceSelect}
      />
    )
  }

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
      sites={sites}
      onSiteClick={handleSiteClick}
      lastUpdated={lastUpdated}
      dataMode={dataMode}
      loadError={loadError}
    />
  )
}

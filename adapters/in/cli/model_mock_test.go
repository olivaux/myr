// adapters/in/cli/model_mock_test.go — mock partagé ModelService (Pattern B — fn-func)
package cli

import "myr/domain/model"

type mockModelSvc struct {
	add                             func(filePath, name, channelID, ownerID string, tags []string) (*model.Model3D, error)
	get                             func(id, channelID string) (*model.Model3D, error)
	list                            func(channelID string) ([]*model.Model3D, error)
	verify                          func(id, channelID string) (bool, error)
	addFull                         func(req model.AddRequest) (*model.Model3D, error)
	remove                          func(id string) error
	addConnection                   func(from, to, label string) (*model.Connection, error)
	addAssemblyLink                 func(fromIfaceID, toIfaceID, label, fromInstanceID, toInstanceID, fastenerAssetID string) (*model.Connection, error)
	removeConnection                func(id string) error
	listConnections                 func() ([]*model.Connection, error)
	getChildren                     func(parentID string) ([]*model.Model3D, error)
	saveThumbnail                   func(assetID, dataURL string) error
	getThumbnail                    func(assetID string) (string, error)
	addInterface                    func(iface *model.AssetInterface) error
	updateInterface                 func(iface *model.AssetInterface) error
	removeInterface                 func(id string) error
	listInterfacesForAsset          func(assetID string) ([]*model.AssetInterface, error)
	getInterface                    func(id string) (*model.AssetInterface, error)
	ensureVirtualSlot               func(assetID string)
	getRefs                         func() (*model.InterfaceRefs, error)
	addRefCategory                  func(cat string) error
	addRefType                      func(cat, typeName string) error
	addRefUnit                      func(cat, unit string) error
	updateAsset                     func(req model.UpdateRequest) (*model.Model3D, error)
	createModule                    func(req model.ModuleRequest) (*model.Model3D, error)
	getModule                       func(id string) (*model.Model3D, error)
	listModules                     func(channelID string) ([]*model.Model3D, error)
	addAssemblyToModule             func(moduleID, connID string) error
	removeAssemblyFromModule        func(moduleID, connID string) error
	submitModule                    func(moduleID, note string) (*model.Model3D, error)
	removeModule                    func(id string) error
	getModuleInterfaces             func(moduleID string) ([]*model.AssetInterface, error)
	addAssetToWorkspace             func(moduleID, assetID string) (*model.Model3D, error)
	removeAssetFromWorkspace        func(moduleID, instanceID string) (*model.Model3D, error)
	updateInstancePosition          func(moduleID, instanceID string, x, y float64) (*model.Model3D, error)
	listLicenses                    func() []*model.License
	getLicense                      func(id string) (*model.License, error)
	checkLicenseCompatibility       func(parentLicenseID, proposedLicenseID string) *model.LicenseCheck
	checkModuleLicenseCompatibility func(componentLicenseIDs []string, proposedProductLicenseID string) *model.LicenseCheck
	connectVirtualToPhysical        func(virtualIfaceID, physicalIfaceID string, popupValues model.AssetInterface, fromInstanceID, toInstanceID string) (*model.Connection, error)
}

func (m *mockModelSvc) Add(filePath, name, channelID, ownerID string, tags []string) (*model.Model3D, error) {
	if m.add != nil {
		return m.add(filePath, name, channelID, ownerID, tags)
	}
	return &model.Model3D{ID: "m-test", Name: name}, nil
}
func (m *mockModelSvc) Get(id, channelID string) (*model.Model3D, error) {
	if m.get != nil {
		return m.get(id, channelID)
	}
	return &model.Model3D{ID: id}, nil
}
func (m *mockModelSvc) List(channelID string) ([]*model.Model3D, error) {
	if m.list != nil {
		return m.list(channelID)
	}
	return nil, nil
}
func (m *mockModelSvc) Verify(id, channelID string) (bool, error) {
	if m.verify != nil {
		return m.verify(id, channelID)
	}
	return true, nil
}
func (m *mockModelSvc) AddFull(req model.AddRequest) (*model.Model3D, error) {
	if m.addFull != nil {
		return m.addFull(req)
	}
	return &model.Model3D{ID: "m-test", Name: req.Name}, nil
}
func (m *mockModelSvc) Remove(id string) error {
	if m.remove != nil {
		return m.remove(id)
	}
	return nil
}
func (m *mockModelSvc) AddConnection(from, to, label string) (*model.Connection, error) {
	if m.addConnection != nil {
		return m.addConnection(from, to, label)
	}
	return &model.Connection{ID: "c-test", From: from, To: to}, nil
}
func (m *mockModelSvc) AddAssemblyLink(fromIfaceID, toIfaceID, label, fromInstanceID, toInstanceID, fastenerAssetID string) (*model.Connection, error) {
	if m.addAssemblyLink != nil {
		return m.addAssemblyLink(fromIfaceID, toIfaceID, label, fromInstanceID, toInstanceID, fastenerAssetID)
	}
	return &model.Connection{ID: "c-test", FromIfaceID: fromIfaceID, ToIfaceID: toIfaceID}, nil
}
func (m *mockModelSvc) RemoveConnection(id string) error {
	if m.removeConnection != nil {
		return m.removeConnection(id)
	}
	return nil
}
func (m *mockModelSvc) ListConnections() ([]*model.Connection, error) {
	if m.listConnections != nil {
		return m.listConnections()
	}
	return nil, nil
}
func (m *mockModelSvc) GetChildren(parentID string) ([]*model.Model3D, error) {
	if m.getChildren != nil {
		return m.getChildren(parentID)
	}
	return nil, nil
}
func (m *mockModelSvc) SaveThumbnail(assetID, dataURL string) error {
	if m.saveThumbnail != nil {
		return m.saveThumbnail(assetID, dataURL)
	}
	return nil
}
func (m *mockModelSvc) GetThumbnail(assetID string) (string, error) {
	if m.getThumbnail != nil {
		return m.getThumbnail(assetID)
	}
	return "", nil
}
func (m *mockModelSvc) AddInterface(iface *model.AssetInterface) error {
	if m.addInterface != nil {
		return m.addInterface(iface)
	}
	iface.ID = "iface-test"
	return nil
}
func (m *mockModelSvc) UpdateInterface(iface *model.AssetInterface) error {
	if m.updateInterface != nil {
		return m.updateInterface(iface)
	}
	return nil
}
func (m *mockModelSvc) RemoveInterface(id string) error {
	if m.removeInterface != nil {
		return m.removeInterface(id)
	}
	return nil
}
func (m *mockModelSvc) ListInterfacesForAsset(assetID string) ([]*model.AssetInterface, error) {
	if m.listInterfacesForAsset != nil {
		return m.listInterfacesForAsset(assetID)
	}
	return nil, nil
}
func (m *mockModelSvc) GetInterface(id string) (*model.AssetInterface, error) {
	if m.getInterface != nil {
		return m.getInterface(id)
	}
	return &model.AssetInterface{ID: id}, nil
}
func (m *mockModelSvc) EnsureVirtualSlot(assetID string) {
	if m.ensureVirtualSlot != nil {
		m.ensureVirtualSlot(assetID)
	}
}
func (m *mockModelSvc) GetRefs() (*model.InterfaceRefs, error) {
	if m.getRefs != nil {
		return m.getRefs()
	}
	return model.DefaultInterfaceRefs(), nil
}
func (m *mockModelSvc) AddRefCategory(cat string) error {
	if m.addRefCategory != nil {
		return m.addRefCategory(cat)
	}
	return nil
}
func (m *mockModelSvc) AddRefType(cat, typeName string) error {
	if m.addRefType != nil {
		return m.addRefType(cat, typeName)
	}
	return nil
}
func (m *mockModelSvc) AddRefUnit(cat, unit string) error {
	if m.addRefUnit != nil {
		return m.addRefUnit(cat, unit)
	}
	return nil
}
func (m *mockModelSvc) UpdateAsset(req model.UpdateRequest) (*model.Model3D, error) {
	if m.updateAsset != nil {
		return m.updateAsset(req)
	}
	return &model.Model3D{ID: req.ID, Name: req.Name}, nil
}
func (m *mockModelSvc) CreateModule(req model.ModuleRequest) (*model.Model3D, error) {
	if m.createModule != nil {
		return m.createModule(req)
	}
	return &model.Model3D{ID: "mod-test", Name: req.Name}, nil
}
func (m *mockModelSvc) GetModule(id string) (*model.Model3D, error) {
	if m.getModule != nil {
		return m.getModule(id)
	}
	return &model.Model3D{ID: id}, nil
}
func (m *mockModelSvc) ListModules(channelID string) ([]*model.Model3D, error) {
	if m.listModules != nil {
		return m.listModules(channelID)
	}
	return nil, nil
}
func (m *mockModelSvc) AddAssemblyToModule(moduleID, connID string) error {
	if m.addAssemblyToModule != nil {
		return m.addAssemblyToModule(moduleID, connID)
	}
	return nil
}
func (m *mockModelSvc) RemoveAssemblyFromModule(moduleID, connID string) error {
	if m.removeAssemblyFromModule != nil {
		return m.removeAssemblyFromModule(moduleID, connID)
	}
	return nil
}
func (m *mockModelSvc) SubmitModule(moduleID, note string) (*model.Model3D, error) {
	if m.submitModule != nil {
		return m.submitModule(moduleID, note)
	}
	return &model.Model3D{ID: moduleID, Status: model.ModuleSubmitted}, nil
}
func (m *mockModelSvc) RemoveModule(id string) error {
	if m.removeModule != nil {
		return m.removeModule(id)
	}
	return nil
}
func (m *mockModelSvc) GetModuleInterfaces(moduleID string) ([]*model.AssetInterface, error) {
	if m.getModuleInterfaces != nil {
		return m.getModuleInterfaces(moduleID)
	}
	return nil, nil
}
func (m *mockModelSvc) AddAssetToWorkspace(moduleID, assetID string) (*model.Model3D, error) {
	if m.addAssetToWorkspace != nil {
		return m.addAssetToWorkspace(moduleID, assetID)
	}
	return &model.Model3D{ID: moduleID, WorkspaceInstances: []model.WorkspaceInstance{{ID: "inst-test", AssetID: assetID}}}, nil
}
func (m *mockModelSvc) RemoveAssetFromWorkspace(moduleID, instanceID string) (*model.Model3D, error) {
	if m.removeAssetFromWorkspace != nil {
		return m.removeAssetFromWorkspace(moduleID, instanceID)
	}
	return &model.Model3D{ID: moduleID}, nil
}
func (m *mockModelSvc) UpdateInstancePosition(moduleID, instanceID string, x, y float64) (*model.Model3D, error) {
	if m.updateInstancePosition != nil {
		return m.updateInstancePosition(moduleID, instanceID, x, y)
	}
	return &model.Model3D{ID: moduleID}, nil
}
func (m *mockModelSvc) ListLicenses() []*model.License {
	if m.listLicenses != nil {
		return m.listLicenses()
	}
	return nil
}
func (m *mockModelSvc) GetLicense(id string) (*model.License, error) {
	if m.getLicense != nil {
		return m.getLicense(id)
	}
	return &model.License{ID: id}, nil
}
func (m *mockModelSvc) CheckLicenseCompatibility(parentLicenseID, proposedLicenseID string) *model.LicenseCheck {
	if m.checkLicenseCompatibility != nil {
		return m.checkLicenseCompatibility(parentLicenseID, proposedLicenseID)
	}
	return &model.LicenseCheck{Compatible: true}
}
func (m *mockModelSvc) CheckModuleLicenseCompatibility(componentLicenseIDs []string, proposedProductLicenseID string) *model.LicenseCheck {
	if m.checkModuleLicenseCompatibility != nil {
		return m.checkModuleLicenseCompatibility(componentLicenseIDs, proposedProductLicenseID)
	}
	return &model.LicenseCheck{Compatible: true}
}
func (m *mockModelSvc) ConnectVirtualToPhysical(virtualIfaceID, physicalIfaceID string, popupValues model.AssetInterface, fromInstanceID, toInstanceID string) (*model.Connection, error) {
	if m.connectVirtualToPhysical != nil {
		return m.connectVirtualToPhysical(virtualIfaceID, physicalIfaceID, popupValues, fromInstanceID, toInstanceID)
	}
	return &model.Connection{ID: "c-test"}, nil
}

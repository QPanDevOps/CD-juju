// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package storage_test

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/apiserver/storage"
	"github.com/juju/juju/apiserver/testing"
	jujutesting "github.com/juju/juju/juju/testing"
	"github.com/juju/juju/state"
	jujustorage "github.com/juju/juju/storage"
	"github.com/juju/juju/storage/pool"
	"github.com/juju/juju/storage/provider"
	"github.com/juju/juju/storage/provider/registry"
)

var (
	testPool = "block"
)

type storageSuite struct {
	// TODO(anastasiamac) mock to remove JujuConnSuite
	jujutesting.JujuConnSuite

	api        *storage.API
	authorizer testing.FakeAuthorizer
}

var _ = gc.Suite(&storageSuite{})

func (s *storageSuite) SetUpTest(c *gc.C) {
	s.JujuConnSuite.SetUpTest(c)
	s.authorizer = testing.FakeAuthorizer{
		Tag: s.AdminUserTag(c),
	}

	setupTestStorageSupport(c, s.State)

	var err error
	s.api, err = storage.NewAPI(s.State, nil, s.authorizer)
	c.Assert(err, jc.ErrorIsNil)
}

func setupTestStorageSupport(c *gc.C, s *state.State) {
	cfg, err := s.EnvironConfig()
	c.Assert(err, jc.ErrorIsNil)

	stsetts := state.NewStateSettings(s)
	poolManager := pool.NewPoolManager(stsetts)
	_, err = poolManager.Create(testPool, provider.LoopProviderType, map[string]interface{}{"it": "works"})
	c.Assert(err, jc.ErrorIsNil)

	registry.RegisterEnvironStorageProviders("someprovider", provider.LoopProviderType)
	registry.RegisterDefaultPool(cfg.Type(), jujustorage.StorageKindBlock, testPool)
}

func makeStorageCons(pool string, size, count uint64) state.StorageConstraints {
	return state.StorageConstraints{Pool: pool, Size: size, Count: count}
}

func createUnitForTest(c *gc.C, s *jujutesting.JujuConnSuite) string {
	ch := s.AddTestingCharm(c, "storage-block")
	storage := map[string]state.StorageConstraints{
		"data": makeStorageCons(testPool, 1024, 1),
	}
	service := s.AddTestingServiceWithStorage(c, "storage-block", ch, storage)
	unit, err := service.AddUnit()
	c.Assert(err, jc.ErrorIsNil)
	err = s.State.AssignUnit(unit, state.AssignCleanEmpty)
	c.Assert(err, jc.ErrorIsNil)

	machineId, err := unit.AssignedMachineId()
	c.Assert(err, jc.ErrorIsNil)

	return machineId
}

func (s *storageSuite) TestShowStorage(c *gc.C) {
	createUnitForTest(c, &s.JujuConnSuite)

	storageTag := "storage-data-0"
	entity := params.Entity{Tag: storageTag}

	found, err := s.api.Show(params.Entities{Entities: []params.Entity{entity}})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(found.Results, gc.HasLen, 1)

	one := found.Results[0]
	c.Assert(one.Error, gc.IsNil)
	c.Assert(one.Result.StorageTag, gc.DeepEquals, "storage-data-0")
	c.Assert(one.Result.OwnerTag, gc.DeepEquals, "unit-storage-block-0")
	c.Assert(one.Result.Kind, gc.DeepEquals, params.StorageKindBlock)
}

func (s *storageSuite) TestShowStorageInvalidId(c *gc.C) {
	storageTag := "foo"
	entity := params.Entity{Tag: storageTag}

	found, err := s.api.Show(params.Entities{Entities: []params.Entity{entity}})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(found.Results, gc.HasLen, 1)
	c.Assert(found.Results[0].Error.Error(), gc.Matches, ".*permission denied*")
}

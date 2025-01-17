package heimdall

import (
	"context"
	"testing"

	"github.com/ledgerwatch/log/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	libcommon "github.com/ledgerwatch/erigon-lib/common"
)

func makeEntities(count uint64) []*Checkpoint {
	var entities []*Checkpoint
	for i := uint64(0); i < count; i++ {
		c := makeCheckpoint(i*256, 256)
		c.Id = CheckpointId(i + 1)
		entities = append(entities, c)
	}
	return entities
}

func makeFetchEntitiesPage(
	entities []*Checkpoint,
) func(ctx context.Context, page uint64, limit uint64) ([]*Checkpoint, error) {
	return func(ctx context.Context, page uint64, limit uint64) ([]*Checkpoint, error) {
		offset := (page - 1) * limit
		totalLen := uint64(len(entities))
		return entities[min(offset, totalLen):min(offset+limit, totalLen)], nil
	}
}

func TestEntityFetcher_FetchAllEntities(t *testing.T) {
	for count := uint64(0); count < 20; count++ {
		testEntityFetcher_FetchAllEntities(t, count, 5)
	}
}

func testEntityFetcher_FetchAllEntities(t *testing.T, count uint64, fetchEntitiesPageLimit uint64) {
	ctx := context.Background()
	logger := log.New()

	expectedEntities := makeEntities(count)
	servedEntities := make([]*Checkpoint, len(expectedEntities))
	copy(servedEntities, expectedEntities)
	libcommon.SliceShuffle(servedEntities)
	fetchEntitiesPage := makeFetchEntitiesPage(servedEntities)

	fetcher := newEntityFetcher[*Checkpoint](
		"fetcher",
		nil,
		nil,
		fetchEntitiesPage,
		fetchEntitiesPageLimit,
		logger,
	)

	actualEntities, err := fetcher.FetchAllEntities(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedEntities, actualEntities)
}

type entityFetcherFetchEntitiesRangeTest struct {
	fetcher entityFetcher[*Checkpoint]

	testRange        ClosedRange
	expectedEntities []*Checkpoint

	ctx    context.Context
	logger log.Logger
}

func newEntityFetcherFetchEntitiesRangeTest(count uint64, withPaging bool, testRange *ClosedRange) entityFetcherFetchEntitiesRangeTest {
	ctx := context.Background()
	logger := log.New()

	if testRange == nil {
		testRange = &ClosedRange{1, count}
	}

	expectedEntities := makeEntities(count)
	fetchEntity := func(ctx context.Context, id int64) (*Checkpoint, error) {
		return expectedEntities[id-1], nil
	}

	fetchEntitiesPage := makeFetchEntitiesPage(expectedEntities)
	if !withPaging {
		fetchEntitiesPage = nil
	}

	fetcher := newEntityFetcher[*Checkpoint](
		"fetcher",
		nil,
		fetchEntity,
		fetchEntitiesPage,
		entityFetcherBatchFetchThreshold,
		logger,
	)

	return entityFetcherFetchEntitiesRangeTest{
		fetcher:          fetcher,
		testRange:        *testRange,
		expectedEntities: expectedEntities,
		ctx:              ctx,
		logger:           logger,
	}
}

func (test entityFetcherFetchEntitiesRangeTest) Run(t *testing.T) {
	actualEntities, err := test.fetcher.FetchEntitiesRange(test.ctx, test.testRange)
	require.NoError(t, err)
	assert.Equal(t, test.expectedEntities[test.testRange.Start-1:test.testRange.End], actualEntities)
}

func TestEntityFetcher_FetchEntitiesRange(t *testing.T) {
	makeTest := newEntityFetcherFetchEntitiesRangeTest
	var many uint64 = entityFetcherBatchFetchThreshold + 1

	t.Run("no paging", makeTest(1, false, nil).Run)
	t.Run("paging few", makeTest(1, true, nil).Run)
	t.Run("paging many", makeTest(many, true, nil).Run)
	t.Run("paging many subrange", makeTest(many, true, &ClosedRange{many / 3, many / 2}).Run)
}

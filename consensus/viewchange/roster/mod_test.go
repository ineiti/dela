package roster

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/consensus/viewchange"
	types "go.dedis.ch/dela/consensus/viewchange/roster/json"
	"go.dedis.ch/dela/crypto/bls"
	"go.dedis.ch/dela/encoding"
	internal "go.dedis.ch/dela/internal/testing"
	"go.dedis.ch/dela/internal/testing/fake"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde/json"
)

func TestMessages(t *testing.T) {
	messages := []proto.Message{
		&Roster{},
	}

	for _, m := range messages {
		internal.CoverProtoMessage(t, m)
	}
}

func TestIterator_Seek(t *testing.T) {
	roster := New(fake.NewAuthority(3, fake.NewSigner)).(roster)
	iter := &iterator{
		roster: &roster,
	}

	iter.Seek(2)
	require.True(t, iter.HasNext())
	iter.Seek(3)
	require.False(t, iter.HasNext())
}

func TestIterator_HasNext(t *testing.T) {
	iter := &iterator{
		roster: &roster{addrs: make([]mino.Address, 3)},
	}

	require.True(t, iter.HasNext())

	iter.index = 1
	require.True(t, iter.HasNext())

	iter.index = 2
	require.True(t, iter.HasNext())

	iter.index = 3
	require.False(t, iter.HasNext())

	iter.index = 10
	require.False(t, iter.HasNext())
}

func TestIterator_GetNext(t *testing.T) {
	iter := &iterator{
		roster: &roster{addrs: make([]mino.Address, 3)},
	}

	for i := 0; i < 3; i++ {
		c := iter.GetNext()
		require.NotNil(t, c)
	}

	require.Equal(t, 3, iter.GetNext())
}

func TestAddressIterator_GetNext(t *testing.T) {
	roster := New(fake.NewAuthority(3, fake.NewSigner)).(roster)
	iter := &addressIterator{
		iterator: &iterator{
			roster: &roster,
		},
	}

	for _, target := range roster.addrs {
		addr := iter.GetNext()
		require.Equal(t, target, addr)
	}

	require.Nil(t, iter.GetNext())
}

func TestPublicKeyIterator_GetNext(t *testing.T) {
	roster := New(fake.NewAuthority(3, fake.NewSigner)).(roster)
	iter := &publicKeyIterator{
		iterator: &iterator{
			roster: &roster,
		},
	}

	for _, target := range roster.pubkeys {
		pubkey := iter.GetNext()
		require.Equal(t, target, pubkey)
	}

	require.Nil(t, iter.GetNext())
}

func TestRoster_Fingerprint(t *testing.T) {
	roster := New(fake.NewAuthority(2, fake.NewSigner)).(roster)

	out := new(bytes.Buffer)
	err := roster.Fingerprint(out)
	require.NoError(t, err)
	require.Equal(t, "\x00\x00\x00\x00\xdf\x01\x00\x00\x00\xdf", out.String())

	roster.addrs[0] = fake.NewBadAddress()
	err = roster.Fingerprint(out)
	require.EqualError(t, err, "couldn't marshal address: fake error")

	roster.addrs[0] = fake.NewAddress(0)
	roster.pubkeys[0] = fake.NewBadPublicKey()
	err = roster.Fingerprint(out)
	require.EqualError(t, err, "couldn't marshal public key: fake error")

	roster.pubkeys[0] = fake.PublicKey{}
	err = roster.Fingerprint(fake.NewBadHash())
	require.EqualError(t, err, "couldn't write address: fake error")

	err = roster.Fingerprint(fake.NewBadHashWithDelay(1))
	require.EqualError(t, err, "couldn't write public key: fake error")
}

func TestRoster_Take(t *testing.T) {
	roster := New(fake.NewAuthority(3, fake.NewSigner))

	roster2 := roster.Take(mino.RangeFilter(1, 2))
	require.Equal(t, 1, roster2.Len())

	roster2 = roster.Take(mino.RangeFilter(1, 3))
	require.Equal(t, 2, roster2.Len())
}

func TestRoster_Apply(t *testing.T) {
	roster := New(fake.NewAuthority(3, fake.NewSigner))
	require.Equal(t, roster, roster.Apply(nil))

	roster2 := roster.Apply(ChangeSet{Remove: []uint32{3, 2, 0}})
	require.Equal(t, roster.Len()-2, roster2.Len())

	roster3 := roster2.Apply(ChangeSet{Add: []Player{{}}})
	require.Equal(t, roster.Len()-1, roster3.Len())
}

func TestRoster_Diff(t *testing.T) {
	roster1 := New(fake.NewAuthority(3, fake.NewSigner))

	roster2 := New(fake.NewAuthority(4, fake.NewSigner))
	diff := roster1.Diff(roster2).(ChangeSet)
	require.Len(t, diff.Add, 1)

	roster3 := New(fake.NewAuthority(2, fake.NewSigner))
	diff = roster1.Diff(roster3).(ChangeSet)
	require.Len(t, diff.Remove, 1)

	roster4 := New(fake.NewAuthority(3, fake.NewSigner)).(roster)
	roster4.addrs[1] = fake.NewAddress(5)
	diff = roster1.Diff(roster4).(ChangeSet)
	require.Equal(t, []uint32{1, 2}, diff.Remove)
	require.Len(t, diff.Add, 2)

	diff = roster1.Diff((viewchange.Authority)(nil)).(ChangeSet)
	require.Equal(t, ChangeSet{}, diff)
}

func TestRoster_Len(t *testing.T) {
	roster := New(fake.NewAuthority(3, fake.NewSigner))
	require.Equal(t, 3, roster.Len())
}

func TestRoster_GetPublicKey(t *testing.T) {
	authority := fake.NewAuthority(3, fake.NewSigner)
	roster := New(authority)

	iter := authority.AddressIterator()
	i := 0
	for iter.HasNext() {
		pubkey, index := roster.GetPublicKey(iter.GetNext())
		require.Equal(t, authority.GetSigner(i).GetPublicKey(), pubkey)
		require.Equal(t, i, index)
		i++
	}

	pubkey, index := roster.GetPublicKey(fake.NewAddress(999))
	require.Equal(t, -1, index)
	require.Nil(t, pubkey)
}

func TestRoster_AddressIterator(t *testing.T) {
	authority := fake.NewAuthority(3, fake.NewSigner)
	roster := New(authority)

	iter := roster.AddressIterator()
	for i := 0; iter.HasNext(); i++ {
		require.Equal(t, authority.GetAddress(i), iter.GetNext())
	}
}

func TestRoster_PublicKeyIterator(t *testing.T) {
	authority := fake.NewAuthority(3, bls.NewSigner)
	roster := New(authority)

	iter := roster.PublicKeyIterator()
	for i := 0; iter.HasNext(); i++ {
		require.Equal(t, authority.GetSigner(i).GetPublicKey(), iter.GetNext())
	}
}

func TestRoster_Pack(t *testing.T) {
	roster := New(fake.NewAuthority(3, fake.NewSigner)).(roster)

	rosterpb, err := roster.Pack(encoding.NewProtoEncoder())
	require.NoError(t, err)
	require.NotNil(t, rosterpb)

	roster.addrs[1] = fake.NewBadAddress()
	_, err = roster.Pack(encoding.NewProtoEncoder())
	require.EqualError(t, err, "couldn't marshal address: fake error")

	_, err = roster.Pack(fake.BadPackAnyEncoder{})
	require.EqualError(t, err, "couldn't pack public key: fake error")
}

func TestRoster_VisitJSON(t *testing.T) {
	roster := New(fake.NewAuthority(1, fake.NewSigner)).(roster)

	ser := json.NewSerializer()

	data, err := ser.Serialize(roster)
	require.NoError(t, err)
	require.Equal(t, `[{"Address":"AAAAAA==","PublicKey":{}}]`, string(data))

	roster.addrs[0] = fake.NewBadAddress()
	_, err = roster.VisitJSON(ser)
	require.EqualError(t, err, "couldn't marshal address: fake error")

	roster.addrs[0] = fake.NewAddress(0)
	_, err = roster.VisitJSON(fake.NewBadSerializer())
	require.EqualError(t, err, "couldn't serialize public key: fake error")
}

func TestRosterFactory_GetAddressFactory(t *testing.T) {
	factory := defaultFactory{
		addressFactory: fake.AddressFactory{},
	}

	require.NotNil(t, factory.GetAddressFactory())
}

func TestRosterFactory_GetPublicKeyFactory(t *testing.T) {
	factory := defaultFactory{
		pubkeyFactory: fake.PublicKeyFactory{},
	}

	require.NotNil(t, factory.GetPublicKeyFactory())
}

func TestRosterFactory_FromProto(t *testing.T) {
	roster := New(fake.NewAuthority(3, fake.NewSigner))
	rosterpb, err := roster.Pack(encoding.NewProtoEncoder())
	require.NoError(t, err)

	rosterany, err := ptypes.MarshalAny(rosterpb)
	require.NoError(t, err)

	factory := NewRosterFactory(fake.AddressFactory{}, fake.PublicKeyFactory{}).(defaultFactory)

	decoded, err := factory.FromProto(rosterpb)
	require.NoError(t, err)
	require.Equal(t, roster.Len(), decoded.Len())

	_, err = factory.FromProto(rosterany)
	require.NoError(t, err)

	_, err = factory.FromProto(nil)
	require.EqualError(t, err, "invalid message type '<nil>'")

	_, err = factory.FromProto(&Roster{Addresses: [][]byte{{}}})
	require.EqualError(t, err, "mismatch array length 1 != 0")

	factory.pubkeyFactory = fake.NewBadPublicKeyFactory()
	_, err = factory.FromProto(rosterpb)
	require.EqualError(t, err, "couldn't decode public key: fake error")
}

func TestRosterFactory_VisitJSON(t *testing.T) {
	factory := NewRosterFactory(fake.AddressFactory{}, fake.PublicKeyFactory{})

	ser := json.NewSerializer()

	var ro roster
	err := ser.Deserialize([]byte(`[{}]`), factory, &ro)
	require.NoError(t, err)

	_, err = factory.VisitJSON(fake.NewBadFactoryInput())
	require.EqualError(t, err, "couldn't deserialize roster: fake error")

	input := fake.FactoryInput{
		Serde:   fake.NewBadSerializer(),
		Message: types.Roster{{}},
	}
	_, err = factory.VisitJSON(input)
	require.EqualError(t, err, "couldn't deserialize public key: fake error")
}

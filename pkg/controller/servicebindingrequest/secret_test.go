package servicebindingrequest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/redhat-developer/service-binding-operator/test/mocks"
)

func assertSecretNamespacedName(t *testing.T, u *unstructured.Unstructured, ns, name string) {
	assert.Equal(t, ns, u.GetNamespace())
	assert.Equal(t, name, u.GetName())
}

func TestSecretNew(t *testing.T) {
	ns := "secret"
	name := "test-secret"

	f := mocks.NewFake(t, ns)

	matchLabels := map[string]string{}

	sbr := mocks.ServiceBindingRequestMock(ns, name, "", "", deploymentsGVR, matchLabels)

	plan := &Plan{Ns: ns, Name: name, CRDDescription: nil, SBR: *sbr}
	data := map[string][]byte{"key": []byte("value")}

	s := NewSecret(f.FakeDynClient(), plan)
	fakeDynClient := f.FakeDynClient()
	retriever := NewRetriever(fakeDynClient, plan, "SERVICE_BINDING")
	t.Run("customEnvParser", func(t *testing.T) {
		customData, err := s.customEnvParser(data, retriever.cache)
		assert.NoError(t, err)
		assert.NotNil(t, customData)
	})

	t.Run("createOrUpdate", func(t *testing.T) {
		u, err := s.createOrUpdate(data)
		assert.NoError(t, err)
		assertSecretNamespacedName(t, u, ns, name)
	})

	t.Run("Delete", func(t *testing.T) {
		err := s.Delete()
		assert.NoError(t, err)
	})

	t.Run("Commit", func(t *testing.T) {
		u, err := s.Commit(data, retriever.cache)
		assert.NoError(t, err)
		assertSecretNamespacedName(t, u, ns, name)
	})

	t.Run("Get", func(t *testing.T) {
		u, _, err := s.Get()
		assert.NoError(t, err)
		assertSecretNamespacedName(t, u, ns, name)
	})
}

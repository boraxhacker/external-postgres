

```shell
kubebuilder init --domain boraxhacker --repo github.com/boraxhacker/external-postgres
kubebuilder create api --group external-postgres --version v1beta1 --kind PostgresDatabase
kubebuilder create api --group external-postgres --version v1beta1 --kind PostgresInstance
```

```shell
make manifests
```

```shell
make install
```

```shell
make run
```
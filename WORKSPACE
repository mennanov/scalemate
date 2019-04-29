load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.18.3/rules_go-0.18.3.tar.gz"],
    sha256 = "86ae934bd4c43b99893fc64be9d9fc684b81461581df7ea8fc291c816f5ee8c5",
)

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

# TODO: replace with http_archive rule once the new release (>0.17.0) is available.
git_repository(
    name = "bazel_gazelle",
    remote = "https://github.com/bazelbuild/bazel-gazelle.git",
    commit = "99f7bcae18d0c84524eca529384723979ce473bc",
)

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

http_archive(
    name = "com_github_atlassian_bazel_tools",
    strip_prefix = "bazel-tools-7d296003f478325b4a933c2b1372426d3a0926f0",
    urls = ["https://github.com/atlassian/bazel-tools/archive/7d296003f478325b4a933c2b1372426d3a0926f0.zip"],
)

load("@com_github_atlassian_bazel_tools//gorevive:deps.bzl", "go_revive_dependencies")

go_revive_dependencies()

git_repository(
    name = "bazel_gomock",
    remote = "https://github.com/jmhodges/bazel_gomock.git",
    commit = "9199dbae087ef6646397ea51738cdc282740501e",
)

# Docker rules.
http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "aed1c249d4ec8f703edddf35cbe9dfaca0b5f5ea6e4cd9e83e99f3b0d1136c3d",
    strip_prefix = "rules_docker-0.7.0",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.7.0.tar.gz"],
)

# This is NOT needed when going through the language lang_image
# "repositories" function(s).
load("@io_bazel_rules_docker//repositories:repositories.bzl", container_repositories = "repositories")

container_repositories()

load("@io_bazel_rules_docker//go:image.bzl", _go_image_repos = "repositories")

_go_image_repos()

# Gazelle generated dependencies below.

go_repository(
    name = "com_github_dgrijalva_jwt_go",
    commit = "3af4c746e1c248ee8491a3e0c6f7a9cd831e95f8",
    importpath = "github.com/dgrijalva/jwt-go",
)

go_repository(
    name = "com_github_golang_mock",
    commit = "837231f7bb377b365da147e5ff6c031b12f0dfaa",
    importpath = "github.com/golang/mock",
)

go_repository(
    name = "com_github_google_uuid",
    commit = "0cd6bf5da1e1c83f8b45653022c74f71af0538a4",
    importpath = "github.com/google/uuid",
)

go_repository(
    name = "com_github_grpc_ecosystem_go_grpc_middleware",
    commit = "cfaf5686ec79ff8344257723b6f5ba1ae0ffeb4d",
    importpath = "github.com/grpc-ecosystem/go-grpc-middleware",
)

go_repository(
    name = "com_github_jinzhu_gorm",
    commit = "8b07437717e71c2ff00602ae19f8353ba10aafbb",
    importpath = "github.com/jinzhu/gorm",
)

go_repository(
    name = "com_github_jinzhu_inflection",
    commit = "04140366298a54a039076d798123ffa108fff46c",
    importpath = "github.com/jinzhu/inflection",
)

go_repository(
    name = "com_github_sirupsen_logrus",
    commit = "dae0fa8d5b0c810a8ab733fbd5510c7cae84eca4",
    importpath = "github.com/sirupsen/logrus",
)

go_repository(
    name = "com_github_streadway_amqp",
    commit = "14f78b41ce6da3d698c2ef2cc8c0ea7ce9e26688",
    importpath = "github.com/streadway/amqp",
)

go_repository(
    name = "com_github_stretchr_testify",
    commit = "34c6fa2dc70986bccbbffcc6130f6920a924b075",
    importpath = "github.com/stretchr/testify",
)

go_repository(
    name = "org_golang_x_crypto",
    commit = "a1f597ede03a7bef967a422b5b3a5bd08805a01e",
    importpath = "golang.org/x/crypto",
)

go_repository(
    name = "com_github_alexflint_go_filemutex",
    commit = "d358565f3c3f5334209f1e80693e4f621650c489",
    importpath = "github.com/alexflint/go-filemutex",
)

go_repository(
    name = "com_github_lib_pq",
    commit = "7aad666537ab32b76f0966145530335f1fed51fd",
    importpath = "github.com/lib/pq",
)

go_repository(
    name = "com_github_mennanov_fieldmask_utils",
    commit = "2d5d5cc5d12379d150bf7a5aa2ada879637cdc83",
    importpath = "github.com/mennanov/fieldmask-utils",
)

go_repository(
    name = "com_github_go_gormigrate_gormigrate",
    commit = "0c6141ae05e70da27c2216945e1c1b2d5ad4aa46",
    importpath = "github.com/go-gormigrate/gormigrate",
)

go_repository(
    name = "com_github_spf13_cobra",
    commit = "ba1052d4cbce7aac421a96de820558f75199ccbc",
    importpath = "github.com/spf13/cobra",
)

go_repository(
    name = "com_github_spf13_pflag",
    commit = "24fa6976df40757dce6aea913e7b81ade90530e1",
    importpath = "github.com/spf13/pflag",
)

go_repository(
    name = "com_github_fsnotify_fsnotify",
    commit = "1485a34d5d5723fea214f5710708e19a831720e4",
    importpath = "github.com/fsnotify/fsnotify",
)

go_repository(
    name = "com_github_hashicorp_hcl",
    commit = "65a6292f0157eff210d03ed1bf6c59b190b8b906",
    importpath = "github.com/hashicorp/hcl",
)

go_repository(
    name = "com_github_magiconair_properties",
    commit = "7757cc9fdb852f7579b24170bcacda2c7471bb6a",
    importpath = "github.com/magiconair/properties",
)

go_repository(
    name = "com_github_mitchellh_go_homedir",
    commit = "af06845cf3004701891bf4fdb884bfe4920b3727",
    importpath = "github.com/mitchellh/go-homedir",
)

go_repository(
    name = "com_github_mitchellh_mapstructure",
    commit = "f15292f7a699fcc1a38a80977f80a046874ba8ac",
    importpath = "github.com/mitchellh/mapstructure",
)

go_repository(
    name = "com_github_pelletier_go_toml",
    commit = "405d48dc28228aa37553612a601116aafda69c1a",
    importpath = "github.com/pelletier/go-toml",
)

go_repository(
    name = "com_github_spf13_afero",
    commit = "f4711e4db9e9a1d3887343acb72b2bbfc2f686f5",
    importpath = "github.com/spf13/afero",
)

go_repository(
    name = "com_github_spf13_cast",
    commit = "8965335b8c7107321228e3e3702cab9832751bac",
    importpath = "github.com/spf13/cast",
)

go_repository(
    name = "com_github_spf13_jwalterweatherman",
    commit = "94f6ae3ed3bceceafa716478c5fbf8d29ca601a1",
    importpath = "github.com/spf13/jwalterweatherman",
)

go_repository(
    name = "com_github_spf13_viper",
    commit = "9e56dacc08fbbf8c9ee2dbc717553c758ce42bc9",
    importpath = "github.com/spf13/viper",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    commit = "51d6538a90f86fe93ac480b35f37b2be17fef232",
    importpath = "gopkg.in/yaml.v2",
)

go_repository(
    name = "com_github_nats_io_go_nats_streaming",
    commit = "512d9079d04064a6b2788d47e9800269d6e32ba8",
    importpath = "github.com/nats-io/go-nats-streaming",
    build_file_proto_mode = "disable",
)

go_repository(
    name = "com_github_nats_io_go_nats",
    commit = "c528ff487513eec69347b5598eb35d91f0a63820",
    importpath = "github.com/nats-io/go-nats",
)

go_repository(
    name = "com_github_nats_io_nuid",
    commit = "3024a71c3cbe30667286099921591e6fcc328230",
    importpath = "github.com/nats-io/nuid",
)

go_repository(
    name = "com_github_nats_io_nkeys",
    commit = "1546a3320a8f195a5b5c84aef8309377c2e411d5",
    importpath = "github.com/nats-io/nkeys",
)

go_repository(
    name = "com_github_jmoiron_sqlx",
    commit = "1d3423c595d749e4613fce663591b44ae539d377",
    importpath = "github.com/jmoiron/sqlx",
)

go_repository(
    name = "com_github_golang_migrate_migrate",
    importpath = "github.com/golang-migrate/migrate/v4",
    sum = "h1:ad8npPhXfv4DV5RFdlpXSz8TQQnjQHBwh28YTfmYmrU=",
    version = "v4.3.0",
)

go_repository(
    name = "com_github_iancoleman_strcase",
    commit = "3605ed457bf7f8caa1371b4fafadadc026673479",
    importpath = "github.com/iancoleman/strcase",
)

go_repository(
    name = "com_github_hashicorp_go_multierror",
    commit = "886a7fbe3eb1c874d46f623bfa70af45f425b3d1",
    importpath = "github.com/hashicorp/go-multierror",
)

go_repository(
    name = "com_github_hashicorp_errwrap",
    commit = "8a6fb523712970c966eefc6b39ed2c5e74880354",
    importpath = "github.com/hashicorp/errwrap",
)

go_repository(
    name = "com_github_go_ozzo_ozzo_validation",
    commit = "2f76ea62300c36e72bd56c804484cb6db53b69a5",
    importpath = "github.com/go-ozzo/ozzo-validation",
)

go_repository(
    name = "com_github_asaskevich_govalidator",
    commit = "f61b66f89f4a311bef65f13e575bcf1a2ffadda6",
    importpath = "github.com/asaskevich/govalidator",
)

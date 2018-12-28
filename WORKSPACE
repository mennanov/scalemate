load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.16.1/rules_go-0.16.1.tar.gz"],
    sha256 = "f5127a8f911468cd0b2d7a141f17253db81177523e4429796e14d429f5444f5f",
)

http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.15.0/bazel-gazelle-0.15.0.tar.gz"],
    sha256 = "6e875ab4b6bf64a38c352887760f21203ab054676d9c1b274963907e0768740d",
)

load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")


git_repository(
    name = "com_github_atlassian_bazel_tools",
    remote = "https://github.com/mennanov/bazel-tools.git",
    commit = "67472091d1276a4cb67c77464531dcbd2a0f3684",
)

# Uncomment once https://github.com/atlassian/bazel-tools/pull/46 is merged.
#http_archive(
#    name = "com_github_atlassian_bazel_tools",
#    strip_prefix = "bazel-tools-96c1e41762781a1f25de2f45e6f0557c9642ef94",
#    urls = ["https://github.com/atlassian/bazel-tools/archive/96c1e41762781a1f25de2f45e6f0557c9642ef94.zip"],
#)

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

load("@com_github_atlassian_bazel_tools//gorevive:deps.bzl", "go_revive_dependencies")

go_revive_dependencies()

git_repository(
    name = "bazel_gomock",
    remote = "https://github.com/jmhodges/bazel_gomock.git",
    commit = "5b73edb74e569ff404b3beffc809d6d9f205e0e4",
)

# Gazelle generated dependencies below.

go_repository(
    name = "com_github_dgrijalva_jwt_go",
    commit = "06ea1031745cb8b3dab3f6a236daf2b0aa468b7e",
    importpath = "github.com/dgrijalva/jwt-go",
)

go_repository(
    name = "com_github_golang_mock",
    commit = "c34cdb4725f4c3844d095133c6e40e448b86589b",
    importpath = "github.com/golang/mock",
)

go_repository(
    name = "com_github_google_uuid",
    commit = "064e2069ce9c359c118179501254f67d7d37ba24",
    importpath = "github.com/google/uuid",
)

go_repository(
    name = "com_github_grpc_ecosystem_go_grpc_middleware",
    commit = "c250d6563d4d4c20252cd865923440e829844f4e",
    importpath = "github.com/grpc-ecosystem/go-grpc-middleware",
)

go_repository(
    name = "com_github_jinzhu_gorm",
    commit = "6ed508ec6a4ecb3531899a69cbc746ccf65a4166",
    importpath = "github.com/jinzhu/gorm",
)

go_repository(
    name = "com_github_jinzhu_inflection",
    commit = "04140366298a54a039076d798123ffa108fff46c",
    importpath = "github.com/jinzhu/inflection",
)

go_repository(
    name = "com_github_sirupsen_logrus",
    commit = "3e01752db0189b9157070a0e1668a620f9a85da2",
    importpath = "github.com/sirupsen/logrus",
)

go_repository(
    name = "com_github_streadway_amqp",
    commit = "70e15c650864f4fc47f5d3c82ea117285480895d",
    importpath = "github.com/streadway/amqp",
)

go_repository(
    name = "com_github_stretchr_testify",
    commit = "f35b8ab0b5a2cef36673838d662e249dd9c94686",
    importpath = "github.com/stretchr/testify",
)

go_repository(
    name = "org_golang_x_crypto",
    commit = "de0752318171da717af4ce24d0a2e8626afaeb11",
    importpath = "golang.org/x/crypto",
)

go_repository(
    name = "com_github_alexflint_go_filemutex",
    commit = "d358565f3c3f5334209f1e80693e4f621650c489",
    importpath = "github.com/alexflint/go-filemutex",
)

go_repository(
    name = "com_github_khaiql_dbcleaner",
    commit = "3485a1ee4b01c233d640dc163757f60a77df9bf8",
    importpath = "github.com/khaiql/dbcleaner",
)

go_repository(
    name = "com_github_lib_pq",
    commit = "90697d60dd844d5ef6ff15135d0203f65d2f53b8",
    importpath = "github.com/lib/pq",
)

go_repository(
    name = "com_github_mennanov_fieldmask_utils",
    commit = "2c600fd80e3ebeb0e660a69c9f7238156e89b51c",
    importpath = "github.com/mennanov/fieldmask-utils",
)

go_repository(
    name = "com_github_mennanov_gormigrate",
    commit = "1776d94fb9d6887916af26d5e43df9586a32522a",
    importpath = "github.com/mennanov/gormigrate",
)

go_repository(
    name = "com_github_spf13_cobra",
    commit = "ef82de70bb3f60c65fb8eebacbb2d122ef517385",
    importpath = "github.com/spf13/cobra",
)

go_repository(
    name = "com_github_spf13_pflag",
    commit = "9a97c102cda95a86cec2345a6f09f55a939babf5",
    importpath = "github.com/spf13/pflag",
)

go_repository(
    name = "in_gopkg_khaiql_dbcleaner_v2",
    commit = "3485a1ee4b01c233d640dc163757f60a77df9bf8",
    importpath = "gopkg.in/khaiql/dbcleaner.v2",
)

go_repository(
    name = "com_github_fsnotify_fsnotify",
    commit = "c2828203cd70a50dcccfb2761f8b1f8ceef9a8e9",
    importpath = "github.com/fsnotify/fsnotify",
)

go_repository(
    name = "com_github_hashicorp_hcl",
    commit = "ef8a98b0bbce4a65b5aa4c368430a80ddc533168",
    importpath = "github.com/hashicorp/hcl",
)

go_repository(
    name = "com_github_magiconair_properties",
    commit = "c2353362d570a7bfa228149c62842019201cfb71",
    importpath = "github.com/magiconair/properties",
)

go_repository(
    name = "com_github_mitchellh_go_homedir",
    commit = "58046073cbffe2f25d425fe1331102f55cf719de",
    importpath = "github.com/mitchellh/go-homedir",
)

go_repository(
    name = "com_github_mitchellh_mapstructure",
    commit = "f15292f7a699fcc1a38a80977f80a046874ba8ac",
    importpath = "github.com/mitchellh/mapstructure",
)

go_repository(
    name = "com_github_pelletier_go_toml",
    commit = "c01d1270ff3e442a8a57cddc1c92dc1138598194",
    importpath = "github.com/pelletier/go-toml",
)

go_repository(
    name = "com_github_spf13_afero",
    commit = "787d034dfe70e44075ccc060d346146ef53270ad",
    importpath = "github.com/spf13/afero",
)

go_repository(
    name = "com_github_spf13_cast",
    commit = "8965335b8c7107321228e3e3702cab9832751bac",
    importpath = "github.com/spf13/cast",
)

go_repository(
    name = "com_github_spf13_jwalterweatherman",
    commit = "7c0cea34c8ece3fbeb2b27ab9b59511d360fb394",
    importpath = "github.com/spf13/jwalterweatherman",
)

go_repository(
    name = "com_github_spf13_viper",
    commit = "907c19d40d9a6c9bb55f040ff4ae45271a4754b9",
    importpath = "github.com/spf13/viper",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    commit = "5420a8b6744d3b0345ab293f6fcba19c978f1183",
    importpath = "gopkg.in/yaml.v2",
)

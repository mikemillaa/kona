#![doc = include_str!("../README.md")]
#![doc(
    html_logo_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/square.png",
    html_favicon_url = "https://raw.githubusercontent.com/op-rs/kona/main/assets/favicon.ico",
    issue_tracker_base_url = "https://github.com/op-rs/kona/issues/"
)]
#![cfg_attr(not(test), warn(unused_crate_dependencies))]
#![cfg_attr(docsrs, feature(doc_cfg, doc_auto_cfg))]

#[macro_use]
extern crate tracing;

mod metrics;
pub use metrics::Metrics;

mod builder;
pub use builder::{Discv5Builder, LocalNode};

mod error;
pub use error::Discv5BuilderError;

mod driver;
pub use driver::Discv5Driver;

mod handler;
pub use handler::{Discv5Handler, HandlerRequest};
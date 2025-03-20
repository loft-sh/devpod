use crate::daemon::DaemonStatus;
use anyhow::anyhow;
use bytes::{Buf, Bytes};
use core::time;
use http::Uri;
use http_body_util::{BodyExt, Empty};
use hyper::{
    body::{Body, Incoming},
    client::conn::http1::SendRequest,
    header,
};
use log::error;
use pin_project_lite::pin_project;
use serde::de::DeserializeOwned;
use std::{
    io,
    path::Path,
    pin::Pin,
    task::{Context, Poll},
};
use tokio::io::AsyncWriteExt;

pub type Request = hyper::Request<axum::body::Body>;
pub type Response = hyper::Response<hyper::body::Incoming>;
#[derive(Debug)]
pub struct Client {
    socket: String,
}
impl Client {
    pub fn new(socket: String) -> Client {
        return Client { socket };
    }

    pub async fn status(&self) -> anyhow::Result<DaemonStatus> {
        let res = self.get::<DaemonStatus>("/status").await?;
        return Ok(res);
    }

    pub async fn proxy(&self, mut req: Request) -> anyhow::Result<Response> {
        let addr = Path::new(&self.socket);
        let handshake_stream = HandshakeStream::connect(&addr).await?;
        let io = TokioIo::new(handshake_stream);

        let (mut sender, conn) = hyper::client::conn::http1::handshake(io).await?;
        tokio::task::spawn(async move {
            if let Err(err) = conn.await {
                error!("Connection failed: {:?}", err);
            }
        });

        let path_and_query = req
            .uri()
            .path_and_query()
            .map(|pq| pq.as_str())
            .unwrap_or("");
        let new_uri = format!("http://localclient.devpod{}", path_and_query);
        *req.uri_mut() = new_uri
            .parse::<Uri>()
            .map_err(|e| anyhow!("failed to parse new URI: {:?}", e))?;

        req.headers_mut().insert(
            header::HOST,
            header::HeaderValue::from_static("sh.loft.devpod.desktop"),
        );

        let res = sender.send_request(req).await?;
        return Ok(res);
    }

    async fn get<T: DeserializeOwned>(&self, target_path: &str) -> anyhow::Result<T> {
        let addr = Path::new(&self.socket);
        let handshake_stream = HandshakeStream::connect(&addr).await?;
        let io = TokioIo::new(handshake_stream);

        let (mut sender, conn) = hyper::client::conn::http1::handshake(io).await?;
        tokio::task::spawn(async move {
            if let Err(err) = conn.await {
                error!("Connection failed: {:?}", err);
            }
        });

        let req = hyper::Request::builder()
            .uri(format!("http://localclient.devpod{}", target_path))
            .header(hyper::header::HOST, "sh.loft.devpod.desktop")
            .body(Empty::<Bytes>::new())?;

        let res = sender.send_request(req).await?;
        if res.status() != http::StatusCode::OK {
            return Err(anyhow!(
                "request to {} failed: {}",
                target_path,
                res.status()
            ));
        }

        let body = res.collect().await?.aggregate();
        let out: T = serde_json::from_reader(body.reader())?;

        return Ok(out);
    }
}

const DEVPOD_PREFIX_BYTE: u8 = 0x01;

pub struct HandshakeStream {
    inner: InnerStream,
}
#[cfg(not(windows))]
type InnerStream = tokio::net::UnixStream;
#[cfg(windows)]
type InnerStream = tokio::net::windows::named_pipe::NamedPipeClient;

impl HandshakeStream {
    pub async fn connect(p: &Path) -> tokio::io::Result<Self> {
        let mut inner: InnerStream;
        #[cfg(not(windows))]
        {
            inner = tokio::net::UnixStream::connect(p).await?;
        }
        #[cfg(windows)]
        {
            inner = tokio::net::windows::named_pipe::ClientOptions::new().open(p)?;
        }
        let mut hs = HandshakeStream { inner };
        // send devpod prefix as first byte
        hs.inner.write_u8(DEVPOD_PREFIX_BYTE).await?;
        Ok(hs)
    }
}

impl tokio::io::AsyncRead for HandshakeStream {
    fn poll_read(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &mut tokio::io::ReadBuf<'_>,
    ) -> Poll<io::Result<()>> {
        Pin::new(&mut self.get_mut().inner).poll_read(cx, buf)
    }
}

impl tokio::io::AsyncWrite for HandshakeStream {
    fn poll_write(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &[u8],
    ) -> Poll<Result<usize, io::Error>> {
        Pin::new(&mut self.get_mut().inner).poll_write(cx, buf)
    }

    fn poll_flush(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Result<(), io::Error>> {
        Pin::new(&mut self.get_mut().inner).poll_flush(cx)
    }

    fn poll_shutdown(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Result<(), io::Error>> {
        Pin::new(&mut self.get_mut().inner).poll_shutdown(cx)
    }
}

// bridge between tokio and hyper, copied from https://github.com/hyperium/hyper/blob/master/benches/support/tokiort.rs#L92
pin_project! {
    #[derive(Debug)]
    pub struct TokioIo<T> {
        #[pin]
        inner: T,
    }
}

impl<T> TokioIo<T> {
    pub fn new(inner: T) -> Self {
        Self { inner }
    }
}

impl<T> hyper::rt::Read for TokioIo<T>
where
    T: tokio::io::AsyncRead,
{
    fn poll_read(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        mut buf: hyper::rt::ReadBufCursor<'_>,
    ) -> Poll<Result<(), std::io::Error>> {
        let n = unsafe {
            let mut tbuf = tokio::io::ReadBuf::uninit(buf.as_mut());
            match tokio::io::AsyncRead::poll_read(self.project().inner, cx, &mut tbuf) {
                Poll::Ready(Ok(())) => tbuf.filled().len(),
                other => return other,
            }
        };

        unsafe {
            buf.advance(n);
        }
        Poll::Ready(Ok(()))
    }
}

impl<T> hyper::rt::Write for TokioIo<T>
where
    T: tokio::io::AsyncWrite,
{
    fn poll_write(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &[u8],
    ) -> Poll<Result<usize, std::io::Error>> {
        tokio::io::AsyncWrite::poll_write(self.project().inner, cx, buf)
    }

    fn poll_flush(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Result<(), std::io::Error>> {
        tokio::io::AsyncWrite::poll_flush(self.project().inner, cx)
    }

    fn poll_shutdown(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
    ) -> Poll<Result<(), std::io::Error>> {
        tokio::io::AsyncWrite::poll_shutdown(self.project().inner, cx)
    }

    fn is_write_vectored(&self) -> bool {
        tokio::io::AsyncWrite::is_write_vectored(&self.inner)
    }

    fn poll_write_vectored(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        bufs: &[std::io::IoSlice<'_>],
    ) -> Poll<Result<usize, std::io::Error>> {
        tokio::io::AsyncWrite::poll_write_vectored(self.project().inner, cx, bufs)
    }
}

impl<T> tokio::io::AsyncRead for TokioIo<T>
where
    T: hyper::rt::Read,
{
    fn poll_read(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        tbuf: &mut tokio::io::ReadBuf<'_>,
    ) -> Poll<Result<(), std::io::Error>> {
        //let init = tbuf.initialized().len();
        let filled = tbuf.filled().len();
        let sub_filled = unsafe {
            let mut buf = hyper::rt::ReadBuf::uninit(tbuf.unfilled_mut());

            match hyper::rt::Read::poll_read(self.project().inner, cx, buf.unfilled()) {
                Poll::Ready(Ok(())) => buf.filled().len(),
                other => return other,
            }
        };

        let n_filled = filled + sub_filled;
        // At least sub_filled bytes had to have been initialized.
        let n_init = sub_filled;
        unsafe {
            tbuf.assume_init(n_init);
            tbuf.set_filled(n_filled);
        }

        Poll::Ready(Ok(()))
    }
}

impl<T> tokio::io::AsyncWrite for TokioIo<T>
where
    T: hyper::rt::Write,
{
    fn poll_write(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &[u8],
    ) -> Poll<Result<usize, std::io::Error>> {
        hyper::rt::Write::poll_write(self.project().inner, cx, buf)
    }

    fn poll_flush(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Result<(), std::io::Error>> {
        hyper::rt::Write::poll_flush(self.project().inner, cx)
    }

    fn poll_shutdown(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
    ) -> Poll<Result<(), std::io::Error>> {
        hyper::rt::Write::poll_shutdown(self.project().inner, cx)
    }

    fn is_write_vectored(&self) -> bool {
        hyper::rt::Write::is_write_vectored(&self.inner)
    }

    fn poll_write_vectored(
        self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        bufs: &[std::io::IoSlice<'_>],
    ) -> Poll<Result<usize, std::io::Error>> {
        hyper::rt::Write::poll_write_vectored(self.project().inner, cx, bufs)
    }
}

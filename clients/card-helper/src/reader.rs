use std::{ffi::CStr, future::Future};

use pcsc;

struct CardFuture<'a> {
    card: pcsc::Card,
    rapdu_buf: [u8; pcsc::MAX_BUFFER_SIZE],
    apdu: &'a [u8],
}

impl Future for CardFuture<'_> {
    type Output = Result<&'static [u8], pcsc::Error>;

    fn poll(self: std::pin::Pin<&mut Self>, cx: &mut std::task::Context) -> std::task::Poll<Self::Output> {
        let this = self.get_mut();
        match this.card.transmit(this.apdu, &mut this.rapdu_buf) {
            Ok(rapdu) => {
                let rapdu = &rapdu[..rapdu.len() - 2];
                std::task::Poll::Ready(Ok(rapdu))
            }
            Err(pcsc::Error::NoSmartcard) => {
                std::task::Poll::Ready(Err(pcsc::Error::NoSmartcard))
            }
            Err(err) => {
                std::task::Poll::Ready(Err(err))
            }
        }
    }
}


pub struct Reader<'a> {
    context: pcsc::Context,
    reader: &'a CStr,
}

impl Reader<'_> {
    pub fn new(reader: &CStr) -> Result<Reader, pcsc::Error> {
        let context = pcsc::Context::establish(pcsc::Scope::User)?;
        Ok(Reader { context, reader })
    }

    pub fn get_first_reader<'a>() -> Result<&'a CStr, pcsc::Error> {
        let context = pcsc::Context::establish(pcsc::Scope::User)?;
        let mut readers_buf = [0; 2048];
        let mut readers = context.list_readers(&mut readers_buf)?;
        let reader = readers.next().ok_or(pcsc::Error::NoReadersAvailable)?;
        Ok(reader)
    }

    async fn connect(&self) -> Result<pcsc::Card, pcsc::Error> {
        self.context.connect(self.reader, pcsc::ShareMode::Shared, pcsc::Protocols::ANY)
    }
}
use pcsc::*;

use std::{ffi::CString, sync::Arc};
use tokio::sync::Mutex;

pub struct AsyncPcsc {
    ctx: Arc<Mutex<pcsc::Context>>,
}

impl AsyncPcsc {
    pub async fn establish() -> Result<Self, pcsc::Error> {
        let context = pcsc::Context::establish(Scope::User)?;
        Ok(AsyncPcsc {
            ctx: Arc::new(Mutex::new(context)),
        })
    }

    pub async fn list_readers(&self) -> Result<Vec<CString>, pcsc::Error> {
        let context = self.ctx.lock().await;
        let readers = context.list_readers_owned()?;
        Ok(readers)
    }

    pub async fn try_connect(&self, reader: &CString) -> Result<pcsc::Card, pcsc::Error> {
        let context = self.ctx.lock().await;
        let card = context.connect(reader, ShareMode::Shared, Protocols::ANY)?;
        Ok(card)
    }

    pub async fn connect(&self, reader: &CString) -> Result<pcsc::Card, pcsc::Error> {
        let context = self.ctx.lock().await;
        loop {
            match context.connect(reader, ShareMode::Shared, Protocols::ANY) {
                Ok(card) => return Ok(card),
                Err(pcsc::Error::NoSmartcard) => {
                    tokio::time::sleep(std::time::Duration::from_secs(1)).await;
                }
                Err(err) => {
                    return Err(err);
                }
            }
        }
    }

    pub async fn transmit(&self, card: &pcsc::Card, apdu: &[u8]) -> Result<Vec<u8>, pcsc::Error> {
        let _ = self.ctx.lock().await;
        let mut rapdu_buf = [0; MAX_BUFFER_SIZE];
        let rapdu = card.transmit(apdu, &mut rapdu_buf)?;
        Ok(rapdu.to_vec())
    }
}
use std::sync::Arc;

use card_helper::reader::AsyncPcsc;
use image::DynamicImage;
use leash_client::{client::LeashClient, user::User};
use rqrr::Point;
use tokio::{
    sync::{mpsc::{self, error::{TryRecvError, TrySendError, SendError}}, Mutex, TryLockError},
    task::yield_now,
};

#[derive(Debug)]
pub enum TaskError<T> {
    Mutex(TryLockError),
    ChannelTrySend(TrySendError<T>),
    ChannelSend(SendError<T>),
    ChannelTryRecev(TryRecvError),
}

impl<T> From<TryLockError> for TaskError<T> {
    fn from(value: TryLockError) -> Self {
        Self::Mutex(value)
    }
}

impl<T> From<TrySendError<T>> for TaskError<T> {
    fn from(value: TrySendError<T>) -> Self {
        Self::ChannelTrySend(value)
    }
}

impl<T> From<SendError<T>> for TaskError<T> {
    fn from(value: SendError<T>) -> Self {
        Self::ChannelSend(value)
    }
}

impl<T> From<TryRecvError> for TaskError<T> {
    fn from(value: TryRecvError) -> Self {
        Self::ChannelTryRecev(value)
    }
}

#[derive(Debug)]
pub struct BackgroundTaskFunction<I, O> {
    input_rx: mpsc::Receiver<I>,
    output_tx: mpsc::Sender<O>,
    rts: Arc<Mutex<bool>>,
}

impl<I, O> BackgroundTaskFunction<I, O> {
    pub async fn recv(&mut self) -> Option<I> {
        *self.rts.lock().await = true;
        self.input_rx.recv().await
    }

    pub fn try_recv(&mut self) -> Result<I, TaskError<I>> {
        *self.rts.try_lock()? = true;
        Ok(self.input_rx.try_recv()?)
    }

    pub async fn ret(&self, input: O) -> Result<(), TaskError<O>> {
        Ok(self.output_tx.send(input).await?)
    }

    pub fn try_ret(&self, input: O) -> Result<(), TaskError<O>> {
        self.output_tx.try_send(input)?;

        Ok(())
    }
}

#[derive(Debug)]
pub struct BackgroundTaskCaller<I, O> {
    input_tx: mpsc::Sender<I>,
    output_rx: mpsc::Receiver<O>,
    rts: Arc<Mutex<bool>>,
}

impl<I, O> BackgroundTaskCaller<I, O> {
    pub async fn call(&self, input: I) {
        let mut v = self.rts.lock().await;
        if *v {
            self.input_tx.send(input).await;
            *v = false;
        }
    }

    pub fn try_call(&self, input: I) -> Result<(), TaskError<I>> {
        let mut v = self.rts.try_lock()?;
        if *v {
            self.input_tx.try_send(input)?;
            *v = false;
        }

        Ok(())
    }

    pub async fn recv(&mut self) -> Option<O> {
        self.output_rx.recv().await
    }

    pub fn try_recv(&mut self) -> Result<O, TaskError<O>> {
        Ok(self.output_rx.try_recv()?)
    }
}

pub fn new_task<I, O>() -> (BackgroundTaskFunction<I, O>, BackgroundTaskCaller<I, O>) {
    let (input_tx, input_rx) = mpsc::channel::<I>(1);
    let (output_tx, output_rx) = mpsc::channel::<O>(1);
    let rts = Arc::new(Mutex::new(false));
    (BackgroundTaskFunction {
        input_rx,
        output_tx,
        rts: rts.clone(),
    }, BackgroundTaskCaller {
        input_tx,
        output_rx,
        rts,
    })
}

pub async fn qr_reader_task(
    mut task: BackgroundTaskFunction<DynamicImage, Option<([Point; 4], String)>>
) -> ! {
    loop {
        match task.recv().await {
            Some(img) => {
                let mut img = rqrr::PreparedImage::prepare(img.to_luma8());
                // Search for grids, without decoding
                let grids = img.detect_grids();

                // Decode the grid
                let mut qr_result = None;
                for g in grids.iter() {
                    match g.decode() {
                        Ok((_meta, content)) => {
                            qr_result = Some((g.bounds, content));
                            break;
                        }
                        Err(_e) => {
                            continue;
                        }
                    }
                }

                task.ret(qr_result).await.unwrap();
            }
            None => {
                continue;
            }
        }
    }
}

pub async fn qr_checkin_task(
    api: LeashClient,
    mut task: BackgroundTaskFunction<String, User>,
) -> ! {
    loop {
        match task.recv().await {
            Some(token) => {
                let checkin = api
                    .get(&format!("api/users/get/checkin/{}", token), None)
                    .await;

                match checkin {
                    Ok(checkin) => {
                        let body = checkin.body;
                        let p = match serde_json::from_str(&body) {
                            Ok(p) => p,
                            Err(_e) => {
                                continue;
                            }
                        };

                        task.ret(p).await.unwrap();
                    }
                    Err(_e) => {
                        continue;
                    }
                }
            }
            None => {
                continue;
            }
        }
    }
}

pub async fn card_task(mut task: BackgroundTaskFunction<(), String>) -> ! {
    let mut ctx = AsyncPcsc::establish().unwrap();
    let readers = ctx.list_readers().await.unwrap();
    ctx.set_reader(&readers[0]);
    loop {
        match task.recv().await {
            Some(_) => {
                let card = ctx.connect().await.unwrap();
                let card_id = match ctx.get_card_id(&card).await {
                    Ok(id) => id,
                    Err(_e) => {
                        continue;
                    }
                };

                task.ret(card_id).await.unwrap();
            }
            None => {
                continue;
            }
        }
    }
}

pub async fn update_task(
    api: LeashClient,
    mut task: BackgroundTaskFunction<(User, String), ()>,
) -> ! {
    loop {
        match task.recv().await {
            Some((user, card_number)) => {
                let card = format!("{{\"card_id\": \"{}\"}}", card_number);
                let result = api
                    .patch(&format!("api/users/{}", user.id), None, &card)
                    .await;

                match result {
                    Ok(_) => {
                        task.ret(()).await.unwrap();
                    }
                    Err(_e) => {
                        continue;
                    }
                }
            }
            None => {
                continue;
            }
        }
    }
}

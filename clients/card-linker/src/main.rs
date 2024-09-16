// // Prevents additional console window on Windows in release, DO NOT REMOVE!!
// #![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

// use std::sync::Arc;

// use keyring::Entry;
// use leash_client::{
//     client::{LeashAuthenticator, LeashClient},
//     user::User,
// };
// use serde::{Deserialize, Serialize};
// use tauri::{async_runtime::Mutex, Error, Result};

// use card_helper::reader::AsyncPcsc;

// const KEYSTORE_SERVICE: &str = "leash-card-linker";

// type Leash = Arc<Mutex<LeashClient>>;
// type ActiveUser = Arc<Mutex<Option<User>>>;
// type Reader = Arc<Mutex<AsyncPcsc>>;

// fn map_keyring_error(e: keyring::Error) -> Error {
//     Error::Io(std::io::Error::new(std::io::ErrorKind::Other, e))
// }

// fn get_keystores() -> Result<(Entry, Entry)> {
//     let apikey_entry = Entry::new(KEYSTORE_SERVICE, "apikey").map_err(map_keyring_error)?;
//     let base_url_entry = Entry::new(KEYSTORE_SERVICE, "base_url").map_err(map_keyring_error)?;

//     Ok((apikey_entry, base_url_entry))
// }

// fn get_leash_api() -> Result<LeashClient> {
//     let apikey;
//     let base_url;

//     match env!("LEASH_SKIP_STORE") {
//         "true" => {
//             apikey = env!("LEASH_APIKEY").to_owned();
//             base_url = env!("LEASH_URL").to_owned();
//         },
//         "false" => {
//             let (apikey_entry, base_url_entry) = get_keystores()?;
//             apikey = apikey_entry.get_password().unwrap_or_default();
//             base_url = base_url_entry.get_password().unwrap_or_default();
//         },
//         _ => unreachable!()
//     }

//     Ok(LeashClient::new(
//         LeashAuthenticator::ApiKey(apikey),
//         base_url,
//     ))
// }

// #[tauri::command]
// async fn set_reader(
//     reader_state: tauri::State<'_, Reader>,
// ) -> Result<()> {
//     let mut ctx = reader_state.lock().await;
//     let readers = ctx.list_readers().await.unwrap();
//     ctx.set_reader(&readers[0]);

//     Ok(())
// }

// #[tauri::command]
// async fn get_card_id(
//     reader_state: tauri::State<'_, Reader>,
// ) -> Result<String> {
//     let ctx = reader_state.lock().await;
//     let card = ctx.connect().await.unwrap();
//     Ok(ctx.get_card_id(&card).await.unwrap())
// }

// #[tauri::command]
// async fn get_checkin(
//     token: String,
//     api_state: tauri::State<'_, Leash>,
//     user_state: tauri::State<'_, ActiveUser>,
// ) -> Result<()> {
//     let checkin = {
//         let api = api_state.lock().await;
//         api.get(&format!("api/users/get/checkin/{}", token), None)
//             .await
//     };

//     match checkin {
//         Ok(checkin) => {
//             let body = checkin.body;
//             let p: User = serde_json::from_str(&body)?;

//             {
//                 let mut user = user_state.lock().await;
//                 *user = Some(p);
//             }

//             Ok(())
//         }
//         Err(e) => Err(Error::Io(std::io::Error::new(std::io::ErrorKind::Other, e))),
//     }
// }
// #[tauri::command]
// async fn get_user(
//     user_state: tauri::State<'_, ActiveUser>,
// ) -> Result<(String, String)> {
//     let user = match user_state.lock().await.clone() {
//         Some(user) => user,
//         None => {
//             return Err(tauri::Error::FailedToExecuteApi(
//                 tauri::api::Error::Command("get_user".to_string()),
//             ))
//         }
//     };

//     Ok((user.name, user.email))
// }

// #[tauri::command]
// async fn set_card(
//     card_number: String,
//     api_state: tauri::State<'_, Leash>,
//     user_state: tauri::State<'_, ActiveUser>,
// ) -> Result<()> {
//     let user = match user_state.lock().await.clone() {
//         Some(user) => user,
//         None => {
//             return Err(tauri::Error::FailedToExecuteApi(
//                 tauri::api::Error::Command("set_card".to_string()),
//             ))
//         }
//     };

//     let checkin = {
//         let card = format!("{{\"card_id\": \"{}\"}}", card_number);
//         let api = api_state.lock().await;
//         api.patch(&format!("api/users/{}", user.id), None, &card)
//             .await
//     };

//     match checkin {
//         Ok(_) => {
//             let mut u = user_state.lock().await;
//             *u = None;
//             Ok(())
//         }
//         Err(e) => Err(Error::Io(std::io::Error::new(std::io::ErrorKind::Other, e))),
//     }
// }

// #[tauri::command]
// async fn clear_user(user_state: tauri::State<'_, ActiveUser>) -> Result<()> {
//     let mut user = user_state.lock().await;
//     *user = None;
//     Ok(())
// }

// #[tauri::command]
// async fn login(base_url: String, apikey: String, state: tauri::State<'_, Leash>) -> Result<()> {
//     let (apikey_entry, base_url_entry) = get_keystores()?;
//     apikey_entry
//         .set_password(&apikey)
//         .map_err(map_keyring_error)?;
//     base_url_entry
//         .set_password(&base_url)
//         .map_err(map_keyring_error)?;

//     let mut api = state.lock().await;
//     api.base_url = base_url;
//     api.authenticator = LeashAuthenticator::ApiKey(apikey);

//     Ok(())
// }

// fn main() {
//     tauri::Builder::default()
//         .manage(Arc::new(Mutex::new(get_leash_api().unwrap())))
//         .manage(Arc::new(Mutex::new(None::<User>)))
//         .manage(Arc::new(Mutex::new(AsyncPcsc::establish().unwrap())))
//         .invoke_handler(tauri::generate_handler![
//             get_checkin,
//             get_user,
//             set_card,
//             set_reader,
//             get_card_id,
//             clear_user,
//             login
//         ])
//         .run(tauri::generate_context!())
//         .expect("error while running tauri application");
// }

#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")] // hide console window on Windows in release
#![allow(rustdoc::missing_crate_level_docs)] // it's an example

use std::time::Duration;

use eframe::egui::{self, Image};
use image::imageops::grayscale;
use image::{DynamicImage, GenericImageView, ImageBuffer, Pixel, Rgb};
use opencv::core::CV_8UC3;
use opencv::imgproc::{cvt_color, cvt_color_def, LineTypes, COLOR_BGR2RGB};
use opencv::prelude::*;
use opencv::videoio::VideoCapture;
use opencv::{highgui, videoio};
use rqrr::Point;
use tokio::sync::mpsc::{Receiver, Sender};
use tokio::sync::{mpsc, oneshot};

#[tokio::main]
async fn main() -> eframe::Result {
    env_logger::init(); // Log to stderr (if you run with `RUST_LOG=debug`).
    let options = eframe::NativeOptions {
        viewport: egui::ViewportBuilder::default().with_inner_size([400.0, 800.0]),
        ..Default::default()
    };

    let cam = videoio::VideoCapture::new(0, videoio::CAP_ANY).unwrap(); // 0 is the default camera
    let opened = videoio::VideoCapture::is_opened(&cam).unwrap();
    if !opened {
        panic!("Unable to open default camera!");
    }

    let (image_tx, mut image_rx) = mpsc::channel::<DynamicImage>(1);
    let (qr_tx, qr_rx) = mpsc::channel::<Option<([Point; 4], String)>>(1);

    let _t1 = tokio::spawn(async move {
        loop {
            match image_rx.recv().await {
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
                            },
                            Err(e) => {
                                continue;
                            },
                        }
                    }

                    qr_tx.send(qr_result).await.unwrap();
                },
                None => todo!(),
            }
        }
    });

    eframe::run_native(
        "Image Viewer",
        options,
        Box::new(|cc| {
            // This gives us image support:
            egui_extras::install_image_loaders(&cc.egui_ctx);
            Ok(Box::new(MyApp { cap: Box::new(cam), qr_last: None, qr_sent: false, img_tx: image_tx, qr_rx: qr_rx }))
        }),
    )
}

struct MyApp {
    cap: Box<VideoCapture>,
    qr_last: Option<[Point; 4]>,
    qr_sent: bool,
    img_tx: Sender<DynamicImage>,
    qr_rx: Receiver<Option<([Point; 4], String)>>,
}

impl eframe::App for MyApp {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        let cam_img;
        let mut mat1 = Mat::default();
        let mut mat2 = Mat::default();
        self.cap.read(&mut mat1).unwrap();
        if mat1.size().unwrap().width > 0 {
            let mut data = Vec::<opencv::core::Point3_<u8>>::new();
            for v in mat1.to_vec_2d().unwrap() {
                data.extend_from_slice(&v);
            }

            cvt_color_def(&mat1, &mut mat2, COLOR_BGR2RGB).unwrap();
            let size = mat2.size().unwrap();
            let slice = unsafe {
                std::slice::from_raw_parts(
                    mat2.ptr(0).unwrap(),
                    (size.width * size.height * (mat2.elem_size().unwrap() as i32)) as usize,
                )
            };
            let img: ImageBuffer<Rgb<u8>, Vec<_>> =
                ImageBuffer::from_vec(size.width as u32, size.height as u32, slice.to_vec())
                    .unwrap();
            cam_img = DynamicImage::from(img);
        } else {
            cam_img = DynamicImage::new(10, 10, image::ColorType::Rgb8);
        }

        if !self.qr_sent {
            self.img_tx.try_send(cam_img).unwrap();
            self.qr_sent = true;
        } else {
            match self.qr_rx.try_recv() {
                Ok(v) => {
                    self.qr_sent = false;
                    match v {
                        Some((loc, content)) => {
                            self.qr_last = Some(loc);
                        },
                        None => {
                            self.qr_last = None;
                        },
                    }
                },
                Err(_) => {},
            }
        }

        if let Some(loc) = self.qr_last {
            let mut pts = Vec::new();

            for pt in loc.iter() {
                pts.push(opencv::core::Point::new(pt.x, pt.y));
            }

            pts.push(pts.get(0).unwrap().clone());

            let color = [1.0, 1.0, 0.0, 0.0];
            for i in 0..pts.len()-1 {
                println!("aaa: {:?}", pts[i]);
                opencv::imgproc::line(&mut mat2, pts.get(i).unwrap().clone(), pts.get(i+1).unwrap().clone(), color.into(), 3, LineTypes::FILLED as i32, 0).unwrap();
            }
        }


        egui::CentralPanel::default().show(ctx, |ui| {
            egui::ScrollArea::both().show(ui, |ui| {
                let size = mat2.size().unwrap();
                let frame_size = [size.width as usize, size.height as usize];
                let slice = unsafe {
                    std::slice::from_raw_parts(
                        mat2.ptr(0).unwrap(),
                        (size.width * size.height * (mat2.elem_size().unwrap() as i32)) as usize,
                    )
                };
                let img = egui::ColorImage::from_rgb(frame_size, slice);

                let texture = ui.ctx().load_texture("frame", img, Default::default());
                // let c = epaint::ColorImage::from_rgb(size, rgb)
                ui.image(&texture);
            });
        });

        ctx.request_repaint();
    }
}

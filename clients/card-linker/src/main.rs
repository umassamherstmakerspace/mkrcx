#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")] // hide console window on Windows in release

mod tasks;
mod api;

use api::get_leash_api;
use eframe::egui::load::SizedTexture;
use eframe::egui::{self, FontId, RichText};
use image::{DynamicImage, GenericImageView, ImageBuffer, Rgb};
use leash_client::user::User;
use opencv::imgproc::{cvt_color_def, LineTypes, COLOR_BGR2RGB};
use opencv::prelude::*;
use opencv::videoio::VideoCapture;
use opencv::videoio;
use tasks::{card_task, new_task, qr_checkin_task, qr_reader_task, update_task, BackgroundTaskCaller};
use rqrr::Point;

#[tokio::main]
async fn main() -> eframe::Result {
    env_logger::init(); // Log to stderr (if you run with `RUST_LOG=debug`).
    let options = eframe::NativeOptions {
        // viewport: egui::ViewportBuilder::default().with_fullscreen(true),
        viewport: egui::ViewportBuilder::default().with_fullscreen(false),
        ..Default::default()
    };

    let cam = videoio::VideoCapture::new(0, videoio::CAP_ANY).unwrap(); // 0 is the default camera
    let opened = videoio::VideoCapture::is_opened(&cam).unwrap();
    if !opened {
        panic!("Unable to open default camera!");
    }

    let api = get_leash_api().unwrap();
    let (qr_reader_fn, qr_reader_caller) = new_task();
    let _t1 = tokio::spawn(qr_reader_task(qr_reader_fn));
    let (qr_checkin_fn, qr_checkin_caller) = new_task();
    let _t2 = tokio::spawn(qr_checkin_task(api.clone(), qr_checkin_fn));
    let (card_reader_fn, card_reader_caller) = new_task();
    let _t3 = tokio::spawn(card_task(card_reader_fn));
    let (user_update_fn, user_update_caller) = new_task();
    let _t4 = tokio::spawn(update_task(api.clone(), user_update_fn));

    eframe::run_native(
        "Image Viewer",
        options,
        Box::new(|cc| {
            // This gives us image support:
            egui_extras::install_image_loaders(&cc.egui_ctx);
            Ok(Box::new(MyApp { cap: Box::new(cam), qr_last: None, qr_reader_caller, qr_checkin_caller, card_reader_caller, user_update_caller }))
        }),
    )
}

struct MyApp {
    cap: Box<VideoCapture>,
    qr_last: Option<[Point; 4]>,
    qr_reader_caller: BackgroundTaskCaller<DynamicImage, Option<([Point; 4], String)>>,
    qr_checkin_caller: BackgroundTaskCaller<String, User>,
    card_reader_caller: BackgroundTaskCaller<(), String>,
    user_update_caller: BackgroundTaskCaller<(User, String), ()>,
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

        // match self.reader_rts.try_lock() {
        //     Ok(mut v) => {
        //         if *v {
        //             self.image_tx.try_send(cam_img).unwrap();
        //             *v = false;
        //         }
        //     },
        //     Err(_) => {},
        // }

        self.qr_reader_caller.try_call(cam_img);

        if let Ok(v) = self.qr_reader_caller.try_recv() {
            match v {
                Some((loc, content)) => {
                    self.qr_last = Some(loc);
                    self.qr_checkin_caller.try_call(content);
                },
                None => {
                    self.qr_last = None;
                },
            }
        }

        if let Ok(v) = self.qr_checkin_caller.try_recv() {
            println!("user {:?}", v);
            self.card_reader_caller.try_call(());
        }

        if let Ok(v) = self.card_reader_caller.try_recv() {
            println!("card {}", v);
        }

        if let Some(loc) = self.qr_last {
            let mut pts = Vec::new();

            for pt in loc.iter() {
                pts.push(opencv::core::Point::new(pt.x, pt.y));
            }

            pts.push(*pts.first().unwrap());

            let color = [12.0, 242.0, 73.0, 255.0];
            for i in 0..pts.len()-1 {
                opencv::imgproc::line(&mut mat2, *pts.get(i).unwrap(), *pts.get(i+1).unwrap(), color.into(), 3, LineTypes::FILLED as i32, 0).unwrap();
            }
        }

        catppuccin_egui::set_theme(ctx, catppuccin_egui::MACCHIATO);
        egui::CentralPanel::default().show(ctx, |ui| {
            ui.vertical_centered(|ui| {
                ui.label(RichText::new("Scan your Check-In QR Code").font(FontId::proportional(40.0)));
                egui::ScrollArea::both().show(ui, |ui| {
                    let mut flip = Mat::default();
                    opencv::core::flip(&mat2, &mut flip, 1).unwrap();
                    let size = flip.size().unwrap();
                    let frame_size = [size.width as usize, size.height as usize];
                    let slice = unsafe {
                        std::slice::from_raw_parts(
                            flip.ptr(0).unwrap(),
                            (size.width * size.height * (flip.elem_size().unwrap() as i32)) as usize,
                        )
                    };
                    let img = egui::ColorImage::from_rgb(frame_size, slice);
    
                    let texture = ui.ctx().load_texture("frame", img, Default::default());
                    let size = texture.size();
                    ui.image(SizedTexture::new(&texture, [480.0 * (size[0] as f32 / size[1] as f32), 480.0]));
                });
            });
        });

        ctx.request_repaint();
    }
}

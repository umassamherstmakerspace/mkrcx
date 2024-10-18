#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")] // hide console window on Windows in release

mod api;
mod tasks;

use std::time::Duration;

use api::get_leash_api;
use eframe::egui::load::SizedTexture;
use eframe::egui::{self, FontId, RichText};
use image::{DynamicImage, ImageBuffer};
use leash_client::user::User;
use nokhwa::pixel_format::RgbFormat;
use nokhwa::utils::{CameraIndex, RequestedFormat, RequestedFormatType};
use nokhwa::Camera;
use rqrr::Point;
use tasks::{
    card_task, new_task, qr_checkin_task, qr_reader_task, update_task, BackgroundTaskCaller,
};
use tokio::time::Instant;

#[tokio::main]
async fn main() -> eframe::Result {
    env_logger::init(); // Log to stderr (if you run with `RUST_LOG=debug`).
    let options = eframe::NativeOptions {
        viewport: egui::ViewportBuilder::default().with_fullscreen(env!("FULLSCREEN").to_ascii_lowercase() == "true"),
        ..Default::default()
    };

    let index = CameraIndex::Index(0); 
    let requested = RequestedFormat::new::<RgbFormat>(RequestedFormatType::AbsoluteHighestFrameRate);
    let mut camera = Camera::new(index, requested).unwrap();
    camera.open_stream().unwrap();

    let api = get_leash_api();
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
            catppuccin_egui::set_theme(&cc.egui_ctx, catppuccin_egui::MACCHIATO);
            Ok(Box::new(App {
                touchscreen: env!("TOUCHSCREEN").to_ascii_lowercase() == "true",
                cap: Box::new(camera),
                state: State::Camera,
                qr_reader_caller,
                qr_checkin_caller,
                card_reader_caller,
                user_update_caller,
            }))
        }),
    )
}

#[derive(Debug, Clone)]
enum State {
    Camera,
    AlreadyLinked {
        user: User,
    },
    ScanCard {
        user: User,
    },
    Linked {
        timeout: Instant,
    }
}

struct App {
    touchscreen: bool,
    cap: Box<Camera>,
    state: State,
    qr_reader_caller: BackgroundTaskCaller<DynamicImage, Option<([Point; 4], String)>>,
    qr_checkin_caller: BackgroundTaskCaller<String, User>,
    card_reader_caller: BackgroundTaskCaller<(), String>,
    user_update_caller: BackgroundTaskCaller<(User, String), ()>,
}

impl eframe::App for App {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        let mut new_state = None;
        match &self.state {
            State::Camera => {
                let frame: nokhwa::Buffer = self.cap.frame().unwrap();
                let cam_img = DynamicImage::from(match self.cap.frame_format() {
                    nokhwa::utils::FrameFormat::NV12 => {
                        let res = frame.resolution();
                    let mut dst_data = vec![0; 3 * res.width() as usize * res.height() as usize];

                    let src_format = dcv_color_primitives::ImageFormat {
                        pixel_format: dcv_color_primitives::PixelFormat::Nv12,
                        color_space: dcv_color_primitives::ColorSpace::Bt601,
                        num_planes: 1,
                    };
                
                    let dst_format = dcv_color_primitives::ImageFormat {
                        pixel_format: dcv_color_primitives::PixelFormat::Rgb,
                        color_space: dcv_color_primitives::ColorSpace::Rgb,
                        num_planes: 1,
                    };
                
                    dcv_color_primitives::convert_image(
                        res.width(),
                        res.height(),
                        &src_format,
                        None,
                        &[frame.buffer()],
                        &dst_format,
                        None,
                        &mut [&mut *dst_data],
                    ).unwrap();

                        ImageBuffer::from_raw(res.width(), res.height(), dst_data).unwrap()
                    },
                    _ => {
                        frame.decode_image::<RgbFormat>().unwrap()
                    }
                });

                let _ = self.qr_reader_caller.try_call(cam_img.clone());

                if let Ok(v) = self.qr_reader_caller.try_recv() {
                    match v {
                        Some((_, content)) => {
                            new_state = Some(State::Camera);
                            let _ = self.qr_checkin_caller.try_call(content);
                        }
                        None => {
                            new_state = Some(State::Camera);
                        }
                    }
                }

                if let Ok(v) = self.qr_checkin_caller.try_recv() {
                    if v.card_id.is_some() {
                        new_state = Some(State::AlreadyLinked { user: v });
                    } else {
                        new_state = Some(State::ScanCard { user: v });
                    }
                    
                }

                egui::CentralPanel::default().show(ctx, |ui| {
                    ui.vertical_centered(|ui| {
                        ui.label(
                            RichText::new("Scan your Check-In QR Code").font(FontId::proportional(40.0)),
                        );

                        ui.add_space(20.0);

                        ui.separator();
                        
                        ui.add_space(20.0);

                        egui::ScrollArea::both().show(ui, |ui| {
                            let frame_size = [cam_img.width() as usize, cam_img.height() as usize];
                            let img = egui::ColorImage::from_rgb(frame_size, &cam_img.as_bytes());

                            let texture = ui.ctx().load_texture("frame", img, Default::default());
                            let size = texture.size();
                            let mut h = ui.available_height();
                            let mut w = h * (size[0] as f32 / size[1] as f32);

                            if w > ui.available_width() {
                                w = ui.available_width();
                                h = w * (size[1] as f32 / size[0] as f32);
                            }
                            ui.image(SizedTexture::new(
                                &texture,
                                [w, h],
                            ));
                        });
                    });
                });
            },
            State::AlreadyLinked { user } => {
                egui::CentralPanel::default().show(ctx, |ui| {
                    ui.vertical_centered(|ui| {
                        ui.label(
                            RichText::new("Card Already Linked").font(FontId::proportional(40.0)),
                        );

                        ui.add_space(20.0);

                        ui.separator();
                        
                        ui.add_space(20.0);

                        ui.label(
                            RichText::new(format!("A card for {} has already been linked, would you like to relink your card?", user.email)).font(FontId::proportional(20.0)),
                        );

                        ui.add_space(40.0);

                        let yes_btn = ui.add_sized([180.0, 60.0], egui::Button::new("Yes"));
                        if yes_btn.clicked() || (self.touchscreen && yes_btn.hovered()) {
                            new_state = Some(State::ScanCard { user: user.clone() });
                        }

                        let no_btn = ui.add_sized([180.0, 60.0], egui::Button::new("No"));
                        if no_btn.clicked() || (self.touchscreen && no_btn.hovered()) {
                            new_state = Some(State::Camera);
                        }
                    });
                });
            },
            State::ScanCard { user } => {
                let _ = self.card_reader_caller.try_call(());
                if let Ok(v) = self.card_reader_caller.try_recv() {
                    let _ = self.user_update_caller.try_call((user.clone(), v));
                }

                if let Ok(_) = self.user_update_caller.try_recv() {
                    new_state = Some(State::Linked { timeout: Instant::now() + Duration::from_secs(5) });
                }

                egui::CentralPanel::default().show(ctx, |ui| {
                    ui.vertical_centered(|ui| {
                        ui.label(
                            RichText::new("Scan Your Card").font(FontId::proportional(40.0)),
                        );

                        ui.add_space(20.0);

                        ui.separator(); 
                    });
                });
            },
            State::Linked { timeout } => {
                if timeout.elapsed() > Duration::ZERO {
                    new_state = Some(State::Camera);
                }

                egui::CentralPanel::default().show(ctx, |ui| {
                    ui.vertical_centered(|ui| {
                        ui.label(
                            RichText::new("Card Successfully Linked").font(FontId::proportional(40.0)),
                        );

                        ui.add_space(20.0);

                        ui.separator();
                        
                        ui.add_space(20.0);

                        let continue_btn = ui.add_sized([180.0, 60.0], egui::Button::new("Continue"));
                        if continue_btn.clicked() || (self.touchscreen && continue_btn.hovered()) {
                            new_state = Some(State::Camera);
                        }
                    });
                });
            },
        }

        if let Some(state) = new_state {
            self.state = state;

            match &self.state {
                State::Camera => {
                    self.cap.frame().unwrap();
                },
                _ => {}
            }
            let _ = self.qr_reader_caller.try_recv();
            let _ = self.qr_checkin_caller.try_recv();
            let _ = self.card_reader_caller.try_recv();
            let _ = self.user_update_caller.try_recv();
        }

        ctx.request_repaint();
    }
}

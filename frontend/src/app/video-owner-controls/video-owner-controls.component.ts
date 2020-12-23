import { Component, Input } from '@angular/core';
import { Router } from '@angular/router';
import { Video } from '../models';
import { NotificationService, VideoService } from '../services';

@Component({
    selector: 'video-owner-controls',
    templateUrl: './video-owner-controls.component.html',
    styleUrls: ['./video-owner-controls.component.scss']
})
export class VideoOwnerControlsComponent {

    @Input() video: Video;

    constructor(
        private router: Router,
        private videoService: VideoService,
        private notification: NotificationService
    ) { }


    deleteVideo() {
        if (confirm(`Are you sure to delete video \"${this.video.title}\"?`)) {
            this.videoService
                .delete(this.video.id)
                .subscribe(() => {
                    this.notification.message("Video deleted succssfully!");
                    this.router.navigate(['']);
                })
        }
    }

    editVideo() {

    }

}

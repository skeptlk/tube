import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { AuthService, VideoService } from '../services';
import { Video } from '../models';

@Component({
    selector: 'video-page',
    templateUrl: './video.component.html',
    styleUrls: ['./video.component.scss']
})
export class VideoComponent implements OnInit {

    video: Video;
    id: number;
    showOwnerControls: boolean;

    constructor(
        public videoService: VideoService,
        public auth: AuthService,
        private route: ActivatedRoute
    ) { }

    ngOnInit(): void {
        this.id = +this.route.snapshot.paramMap.get('id');
        this.videoService
            .getInfo(this.id)
            .subscribe(vid => {
                this.video = vid;
                this.showOwnerControls = (vid.userId === this.auth.currentUserValue.id);
            });
    }

    toggleLike() {
        if (this.video.isLiked) {
            this.videoService.removeLike();
            this.video.isLiked = false;
            this.video.likes--;
        } else {
            this.videoService.like();
            this.video.likes++;
            this.video.isLiked = true;
            if (this.video.isDisliked) {
                this.video.dislikes--;
                this.video.isDisliked = false;
            }
        }
        
    }

    toggleDislike() {
        if (this.video.isDisliked) {
            this.videoService.removeDislike();
            this.video.isDisliked = false;
            this.video.dislikes--;
        } else {
            this.videoService.dislike();
            this.video.dislikes++;
            this.video.isDisliked = true;
            if (this.video.isLiked) {
                this.video.likes--;
                this.video.isLiked = false;
            }
        }
    }

}

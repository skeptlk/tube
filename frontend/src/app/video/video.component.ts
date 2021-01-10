import { Component, OnInit } from '@angular/core';
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
    categories: string;
    id: number;
    showOwnerControls: boolean;
    isLiked: boolean;
    isDisliked: boolean;

    constructor(
        public videoService: VideoService,
        public auth: AuthService,
        private route: ActivatedRoute
    ) { }

    ngOnInit(): void {
        this.id = +this.route.snapshot.paramMap.get('id');
        this.videoService.getInfo(this.id)
            .subscribe(vid => {
                this.video = vid;
                this.categories = this.video.categories.map(c => c.title).join(', ');
                this.showOwnerControls = (vid.userId === this.auth.currentUserValue?.id);
            });
        if (this.auth.isAuthorized) {
            this.videoService.likeInfo(this.id)
                .subscribe(resp => {
                    this.isLiked = resp.liked;
                    this.isDisliked = resp.disliked;
                });
        }
    }

    toggleLike() {
        if (!this.auth.isAuthorized) {
            return;
        }
        if (this.isLiked) {
            this.videoService.removeLike(this.video.id).toPromise();
            this.isLiked = false;
            this.video.likes--;
        } else {
            this.videoService.like(this.video.id).toPromise();
            this.video.likes++;
            this.isLiked = true;
            if (this.isDisliked) {
                this.video.dislikes--;
                this.isDisliked = false;
            }
        }
    }

    toggleDislike() {
        if (!this.auth.isAuthorized) {
            return;
        }
        if (this.isDisliked) {
            this.videoService.removeDislike(this.video.id).toPromise();
            this.isDisliked = false;
            this.video.dislikes--;
        } else {
            this.videoService.dislike(this.video.id).toPromise();
            this.video.dislikes++
            this.isDisliked = true;
            if (this.isLiked) {
                this.video.likes--;
                this.isLiked = false;
            }
        }
    }

}

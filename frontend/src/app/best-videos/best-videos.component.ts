import { Component, OnInit } from '@angular/core';
import { Video } from '../models';
import { VideoService } from '../services';

@Component({
    selector: 'best-videos',
    templateUrl: './best-videos.component.html',
    styleUrls: ['./best-videos.component.scss']
})
export class BestVideosComponent implements OnInit {

    videos: Video[] = [];

    constructor(public vidService: VideoService) 
    {  }

    ngOnInit(): void {
        this.vidService.getBestVideos().subscribe(vids => {
            this.videos = vids;
        });
    }

}

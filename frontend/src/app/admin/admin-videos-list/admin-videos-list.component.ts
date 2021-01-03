import { Component, OnInit } from '@angular/core';
import { User, Video } from 'src/app/models';
import { AdminService } from 'src/app/services';

@Component({
    selector: 'admin-videos-list',
    templateUrl: './admin-videos-list.component.html'
})
export class AdminVideosListComponent implements OnInit {
    videos: Video[] = [];
    pages: number = 1;
    currentPage: number = 0;
    total: number;
    pageSize: number = 10;

    constructor(private adminService: AdminService) 
    { }

    ngOnInit(): void {
        this.adminService
            .getVideos(0, this.pageSize)
            .subscribe(resp => { 
                console.log("Resp: ", resp);
                
                this.videos = resp.videos; 
                this.total = resp.total;
                this.pages = Math.ceil(resp.total / this.pageSize);
            });
    }

    switchPage(page: number): void {
        const limit = this.pageSize;
        this.currentPage = page;
        this.adminService
            .getUsers(page*limit, limit)
            .subscribe(resp => { this.videos = resp.videos; });
    }

    async deleteVideo(id: number) {
        await this.adminService.deleteVideo(id).toPromise();
        this.switchPage(this.currentPage); // reload
    }
}

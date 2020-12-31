import { Component, Input, OnInit } from '@angular/core';
import { CommentsService } from '../services';
import { Comment, User } from '../models';

@Component({
    selector: 'video-comments',
    templateUrl: './comments.component.html',
    styleUrls: ['./comments.component.scss']
})
export class CommentsComponent implements OnInit {
    @Input() videoId: number;

    comments: Comment[] = [];
    newComment: Comment = new Comment();

    constructor(public commentsService: CommentsService) { }

    ngOnInit() {
        this.commentsService
            .list(this.videoId)
            .subscribe(data => this.comments = data);
    }

    submitComment() {
        this.commentsService
            .create(this.newComment.text, this.videoId)
            .subscribe(comm => {
                if (!this.commentsService.replyTo) {
                    this.comments.push(comm);
                }
                this.commentsService.replyTo = undefined;
            })

        this.newComment = new Comment();
    }

}
